use std::collections::{HashMap, HashSet};
use std::error::Error;
use std::fmt;
use std::fs;
use std::io;
use std::path::{Path, PathBuf};
use std::process::ExitStatus;

use super::{MavenEnvironment, ProjectDetectionEvent, ProjectEnvironment, ProjectType, maven};
use crate::java::{JdkDiscoveryError, JdkInstallation, parse_java_major};

const POM_FILE_NAME: &str = "pom.xml";

/// Error raised while detecting a project's build environment.
#[derive(Debug)]
#[non_exhaustive]
pub enum ProjectDetectionError {
    NotMavenProject {
        project_dir: PathBuf,
    },
    FileAccess {
        path: PathBuf,
        source: io::Error,
    },
    InvalidPom {
        path: PathBuf,
        message: String,
    },
    JavaVersionNotFound {
        pom_path: PathBuf,
    },
    JdkDiscovery {
        major_version: u32,
        source: JdkDiscoveryError,
    },
    MavenWrapperVersionNotFound {
        properties_path: PathBuf,
    },
    MavenWrapperNotFound {
        project_dir: PathBuf,
    },
    MavenCommandUnavailable {
        source: io::Error,
    },
    MavenCommandFailed {
        status: ExitStatus,
    },
    MavenVersionNotFound,
}

impl fmt::Display for ProjectDetectionError {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::NotMavenProject { project_dir } => write!(
                formatter,
                "{} is not a Maven project: {POM_FILE_NAME} was not found",
                project_dir.display()
            ),
            Self::FileAccess { path, source } => {
                write!(formatter, "could not read {}: {source}", path.display())
            }
            Self::InvalidPom { path, message } => {
                write!(formatter, "invalid Maven POM {}: {message}", path.display())
            }
            Self::JavaVersionNotFound { pom_path } => write!(
                formatter,
                "could not determine the Java version from {} (set java.version, maven.compiler.release, maven.compiler.target, or compiler/toolchains plugin configuration)",
                pom_path.display()
            ),
            Self::JdkDiscovery {
                major_version,
                source,
            } => write!(formatter, "could not resolve JDK {major_version}: {source}"),
            Self::MavenWrapperVersionNotFound { properties_path } => write!(
                formatter,
                "could not determine the Maven version from {}",
                properties_path.display()
            ),
            Self::MavenWrapperNotFound { project_dir } => write!(
                formatter,
                "Maven Wrapper properties exist, but no executable wrapper was found in {}",
                project_dir.display()
            ),
            Self::MavenCommandUnavailable { source } => write!(
                formatter,
                "could not run 'mvn --version'; add Maven Wrapper or install Maven: {source}"
            ),
            Self::MavenCommandFailed { status } => {
                write!(formatter, "'mvn --version' failed with status {status}")
            }
            Self::MavenVersionNotFound => write!(
                formatter,
                "could not determine the Maven version from 'mvn --version' output"
            ),
        }
    }
}

impl Error for ProjectDetectionError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::FileAccess { source, .. } | Self::MavenCommandUnavailable { source } => {
                Some(source)
            }
            Self::JdkDiscovery { source, .. } => Some(source),
            _ => None,
        }
    }
}

pub(super) fn detect<F>(
    project_dir: &Path,
    observer: &mut F,
) -> Result<ProjectEnvironment, ProjectDetectionError>
where
    F: FnMut(ProjectDetectionEvent),
{
    observer(ProjectDetectionEvent::InspectingProject {
        project_dir: project_dir.to_owned(),
    });

    let pom_path = project_dir.join(POM_FILE_NAME);
    if !pom_path.is_file() {
        return Err(ProjectDetectionError::NotMavenProject {
            project_dir: project_dir.to_owned(),
        });
    }

    observer(ProjectDetectionEvent::ReadingJavaRequirements {
        pom_path: pom_path.clone(),
    });
    let java_version = detect_java_version(&pom_path)?;
    observer(ProjectDetectionEvent::SearchingForJdk {
        major_version: java_version,
    });
    let java = JdkInstallation::discover(java_version).map_err(|source| {
        ProjectDetectionError::JdkDiscovery {
            major_version: java_version,
            source,
        }
    })?;
    observer(ProjectDetectionEvent::JdkDetected {
        major_version: java.major_version(),
        home: java.home().to_owned(),
    });
    let maven = detect_maven(project_dir, &java, observer)?;
    Ok(ProjectEnvironment {
        project_type: ProjectType::Maven,
        java,
        maven,
    })
}

fn detect_maven<F>(
    project_dir: &Path,
    java: &JdkInstallation,
    observer: &mut F,
) -> Result<MavenEnvironment, ProjectDetectionError>
where
    F: FnMut(ProjectDetectionEvent),
{
    let wrapper_properties = maven::wrapper_properties_path(project_dir);
    if wrapper_properties.is_file() {
        if maven::find_wrapper(project_dir).is_none() {
            return Err(ProjectDetectionError::MavenWrapperNotFound {
                project_dir: project_dir.to_owned(),
            });
        }
        observer(ProjectDetectionEvent::ReadingMavenWrapper {
            properties_path: wrapper_properties.clone(),
        });
        let version = maven::read_wrapper_version(&wrapper_properties)
            .map_err(|source| ProjectDetectionError::FileAccess {
                path: wrapper_properties.clone(),
                source,
            })?
            .ok_or({
                ProjectDetectionError::MavenWrapperVersionNotFound {
                    properties_path: wrapper_properties,
                }
            })?;
        observer(ProjectDetectionEvent::MavenDetected {
            version: version.clone(),
            uses_wrapper: true,
        });
        return Ok(MavenEnvironment {
            version,
            uses_wrapper: true,
            settings_profile: None,
        });
    }

    observer(ProjectDetectionEvent::MavenWrapperUnavailable);
    let executable = maven::find_maven_on_path().ok_or_else(|| {
        ProjectDetectionError::MavenCommandUnavailable {
            source: io::Error::new(io::ErrorKind::NotFound, "Maven is not available on PATH"),
        }
    })?;
    let output =
        maven::run_maven_version(&executable, project_dir, java.home()).map_err(|error| {
            ProjectDetectionError::MavenCommandUnavailable {
                source: error.command_source(),
            }
        })?;
    if !output.status.success() {
        return Err(ProjectDetectionError::MavenCommandFailed {
            status: output.status,
        });
    }

    let combined = format!(
        "{}\n{}",
        String::from_utf8_lossy(&output.stdout),
        String::from_utf8_lossy(&output.stderr)
    );
    let version = maven::parse_maven_version_output(&combined)
        .ok_or(ProjectDetectionError::MavenVersionNotFound)?;
    observer(ProjectDetectionEvent::MavenDetected {
        version: version.clone(),
        uses_wrapper: false,
    });
    Ok(MavenEnvironment {
        version,
        uses_wrapper: false,
        settings_profile: None,
    })
}

fn detect_java_version(pom_path: &Path) -> Result<u32, ProjectDetectionError> {
    let mut model = PomModel::default();
    load_pom_model(pom_path, &mut model, &mut HashSet::new())?;

    let candidates = [
        property(&model, "maven.compiler.release"),
        latest(&model.compiler_releases),
        latest(&model.toolchain_versions),
        property(&model, "java.version"),
        property(&model, "maven.compiler.target"),
        latest(&model.compiler_targets),
        property(&model, "maven.compiler.source"),
        latest(&model.compiler_sources),
    ];
    for value in candidates.into_iter().flatten() {
        if let Some(version) =
            resolve_value(value, &model.properties).and_then(|value| parse_java_major(&value))
        {
            return Ok(version);
        }
    }

    Err(ProjectDetectionError::JavaVersionNotFound {
        pom_path: pom_path.to_owned(),
    })
}

#[derive(Default)]
struct PomModel {
    properties: HashMap<String, String>,
    compiler_releases: Vec<String>,
    compiler_targets: Vec<String>,
    compiler_sources: Vec<String>,
    toolchain_versions: Vec<String>,
}

fn property<'a>(model: &'a PomModel, name: &str) -> Option<&'a str> {
    model.properties.get(name).map(String::as_str)
}

fn latest(values: &[String]) -> Option<&str> {
    values.last().map(String::as_str)
}

fn load_pom_model(
    pom_path: &Path,
    model: &mut PomModel,
    visited: &mut HashSet<PathBuf>,
) -> Result<(), ProjectDetectionError> {
    let canonical_path =
        fs::canonicalize(pom_path).map_err(|source| ProjectDetectionError::FileAccess {
            path: pom_path.to_owned(),
            source,
        })?;
    if !visited.insert(canonical_path) {
        return Ok(());
    }

    let xml = read_to_string(pom_path)?;
    let document =
        roxmltree::Document::parse(&xml).map_err(|error| ProjectDetectionError::InvalidPom {
            path: pom_path.to_owned(),
            message: error.to_string(),
        })?;
    let project = document.root_element();
    if project.tag_name().name() != "project" {
        return Err(ProjectDetectionError::InvalidPom {
            path: pom_path.to_owned(),
            message: "document root is not <project>".to_owned(),
        });
    }

    if let Some(parent) = child(project, "parent") {
        let relative_path = match child(parent, "relativePath") {
            Some(relative_path) => relative_path.text().map(str::trim).unwrap_or("").to_owned(),
            None => "../pom.xml".to_owned(),
        };
        if !relative_path.is_empty() {
            let parent_path = pom_path
                .parent()
                .unwrap_or_else(|| Path::new("."))
                .join(relative_path);
            if parent_path.is_file() {
                load_pom_model(&parent_path, model, visited)?;
            }
        }
    }

    if let Some(properties) = child(project, "properties") {
        for property in properties.children().filter(|node| node.is_element()) {
            if let Some(value) = text(property) {
                model
                    .properties
                    .insert(property.tag_name().name().to_owned(), value);
            }
        }
    }

    collect_plugin_versions(project, model);
    Ok(())
}

fn collect_plugin_versions(project: roxmltree::Node<'_, '_>, model: &mut PomModel) {
    let Some(build) = child(project, "build") else {
        return;
    };
    let plugins = child(build, "plugins")
        .into_iter()
        .flat_map(|plugins| plugins.children())
        .filter(|node| node.is_element() && node.tag_name().name() == "plugin");
    for plugin in plugins {
        let artifact_id = child(plugin, "artifactId").and_then(text);
        let Some(artifact_id) = artifact_id else {
            continue;
        };

        match artifact_id.as_str() {
            "maven-compiler-plugin" => {
                for configuration in plugin
                    .descendants()
                    .filter(|node| node.is_element() && node.tag_name().name() == "configuration")
                {
                    if let Some(value) = child(configuration, "release").and_then(text) {
                        model.compiler_releases.push(value);
                    }
                    if let Some(value) = child(configuration, "target").and_then(text) {
                        model.compiler_targets.push(value);
                    }
                    if let Some(value) = child(configuration, "source").and_then(text) {
                        model.compiler_sources.push(value);
                    }
                }
            }
            "maven-toolchains-plugin" => {
                for configuration in plugin
                    .descendants()
                    .filter(|node| node.is_element() && node.tag_name().name() == "configuration")
                {
                    if let Some(value) = configuration
                        .descendants()
                        .find(|node| node.is_element() && node.tag_name().name() == "version")
                        .and_then(text)
                    {
                        model.toolchain_versions.push(value);
                    }
                }
            }
            _ => {}
        }
    }
}

fn child<'document, 'input>(
    node: roxmltree::Node<'document, 'input>,
    name: &str,
) -> Option<roxmltree::Node<'document, 'input>> {
    node.children()
        .find(|child| child.is_element() && child.tag_name().name() == name)
}

fn text(node: roxmltree::Node<'_, '_>) -> Option<String> {
    node.text()
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .map(str::to_owned)
}

fn resolve_value(value: &str, properties: &HashMap<String, String>) -> Option<String> {
    let mut value = value.trim().to_owned();
    let mut seen = HashSet::new();
    loop {
        let Some(property_name) = value
            .strip_prefix("${")
            .and_then(|value| value.strip_suffix('}'))
        else {
            return Some(value);
        };
        if !seen.insert(property_name.to_owned()) {
            return None;
        }
        value = properties.get(property_name)?.trim().to_owned();
    }
}

fn read_to_string(path: &Path) -> Result<String, ProjectDetectionError> {
    fs::read_to_string(path).map_err(|source| ProjectDetectionError::FileAccess {
        path: path.to_owned(),
        source,
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parses_java_major_versions() {
        assert_eq!(parse_java_major("1.8"), Some(8));
        assert_eq!(parse_java_major("11"), Some(11));
        assert_eq!(parse_java_major("17.0.2"), Some(17));
        assert_eq!(parse_java_major("[21,)"), Some(21));
        assert_eq!(parse_java_major("not-a-version"), None);
    }

    #[test]
    fn parses_maven_versions() {
        assert_eq!(
            maven::parse_wrapper_maven_version(
                "distributionUrl=https\\://repo.maven.apache.org/maven2/org/apache/maven/apache-maven/3.9.9/apache-maven-3.9.9-bin.zip"
            ),
            Some("3.9.9".to_owned())
        );
        assert_eq!(
            maven::parse_wrapper_maven_version(
                "distributionUrl=https://example.test/apache-maven-4.0.0-rc-4-bin.zip"
            ),
            Some("4.0.0-rc-4".to_owned())
        );
        assert_eq!(
            maven::parse_maven_version_output(
                "Apache Maven 3.9.11 (abcdef)\nMaven home: /opt/maven"
            ),
            Some("3.9.11".to_owned())
        );
    }

    #[test]
    fn detects_property_references_and_parent_poms() {
        let directory = tempfile::tempdir().unwrap();
        let parent_directory = directory.path().join("parent");
        let project_directory = directory.path().join("project");
        fs::create_dir_all(&parent_directory).unwrap();
        fs::create_dir_all(&project_directory).unwrap();
        fs::write(
            parent_directory.join(POM_FILE_NAME),
            r#"<project><properties><java.release>17</java.release></properties></project>"#,
        )
        .unwrap();
        fs::write(
            project_directory.join(POM_FILE_NAME),
            r#"<project>
                <parent><relativePath>../parent/pom.xml</relativePath></parent>
                <properties><maven.compiler.release>${java.release}</maven.compiler.release></properties>
            </project>"#,
        )
            .unwrap();

        assert_eq!(
            detect_java_version(&project_directory.join(POM_FILE_NAME)).unwrap(),
            17
        );
    }

    #[test]
    fn honors_an_empty_parent_relative_path() {
        let directory = tempfile::tempdir().unwrap();
        let project_directory = directory.path().join("project");
        fs::create_dir_all(&project_directory).unwrap();
        fs::write(
            directory.path().join(POM_FILE_NAME),
            r#"<project><properties><java.version>8</java.version></properties></project>"#,
        )
        .unwrap();
        let pom = project_directory.join(POM_FILE_NAME);
        fs::write(
            &pom,
            r#"<project>
                <parent>
                    <groupId>example</groupId><artifactId>parent</artifactId><version>1</version>
                    <relativePath/>
                </parent>
            </project>"#,
        )
        .unwrap();

        assert!(matches!(
            detect_java_version(&pom),
            Err(ProjectDetectionError::JavaVersionNotFound { .. })
        ));
    }

    #[test]
    fn detects_compiler_plugin_configuration() {
        let directory = tempfile::tempdir().unwrap();
        let pom = directory.path().join(POM_FILE_NAME);
        fs::write(
            &pom,
            r#"<project xmlns="http://maven.apache.org/POM/4.0.0">
                <build><plugins><plugin>
                    <artifactId>maven-compiler-plugin</artifactId>
                    <configuration><release>11</release></configuration>
                </plugin></plugins></build>
            </project>"#,
        )
        .unwrap();

        assert_eq!(detect_java_version(&pom).unwrap(), 11);
    }

    #[test]
    fn prefers_compiler_release_over_generic_java_version() {
        let directory = tempfile::tempdir().unwrap();
        let pom = directory.path().join(POM_FILE_NAME);
        fs::write(
            &pom,
            r#"<project>
                <properties><java.version>17</java.version></properties>
                <build><plugins><plugin>
                    <artifactId>maven-compiler-plugin</artifactId>
                    <executions><execution><configuration><release>11</release></configuration></execution></executions>
                </plugin></plugins></build>
            </project>"#,
        )
            .unwrap();

        assert_eq!(detect_java_version(&pom).unwrap(), 11);
    }

    #[test]
    fn ignores_compiler_configuration_in_inactive_profiles() {
        let directory = tempfile::tempdir().unwrap();
        let pom = directory.path().join(POM_FILE_NAME);
        fs::write(
            &pom,
            r#"<project>
                <properties><java.version>17</java.version></properties>
                <profiles><profile><id>legacy</id><build><plugins><plugin>
                    <artifactId>maven-compiler-plugin</artifactId>
                    <configuration><release>8</release></configuration>
                </plugin></plugins></build></profile></profiles>
            </project>"#,
        )
        .unwrap();

        assert_eq!(detect_java_version(&pom).unwrap(), 17);
    }

    #[test]
    fn detects_a_maven_wrapper_project() {
        let directory = tempfile::tempdir().unwrap();
        fs::create_dir_all(directory.path().join(".mvn/wrapper")).unwrap();
        fs::write(
            directory.path().join(POM_FILE_NAME),
            r#"<project><properties><java.version>1.8</java.version></properties></project>"#,
        )
        .unwrap();
        let wrapper = directory
            .path()
            .join(if cfg!(windows) { "mvnw.cmd" } else { "mvnw" });
        fs::write(&wrapper, "").unwrap();
        #[cfg(unix)]
        {
            use std::os::unix::fs::PermissionsExt;
            fs::set_permissions(&wrapper, fs::Permissions::from_mode(0o755)).unwrap();
        }
        fs::write(
            directory
                .path()
                .join(".mvn/wrapper/maven-wrapper.properties"),
            "distributionUrl=https://example.test/apache-maven-3.9.9-bin.zip\n",
        )
        .unwrap();

        assert_eq!(
            detect_java_version(&directory.path().join(POM_FILE_NAME)).unwrap(),
            8
        );
        let mut events = Vec::new();
        let java = JdkInstallation::recorded(8, directory.path().join("jdk-8"));
        let maven = detect_maven(directory.path(), &java, &mut |event| events.push(event)).unwrap();
        assert_eq!(maven.version(), "3.9.9");
        assert!(maven.uses_wrapper());
        assert_eq!(
            events,
            vec![
                ProjectDetectionEvent::ReadingMavenWrapper {
                    properties_path: directory
                        .path()
                        .join(".mvn/wrapper/maven-wrapper.properties"),
                },
                ProjectDetectionEvent::MavenDetected {
                    version: "3.9.9".to_owned(),
                    uses_wrapper: true,
                },
            ]
        );
    }

    #[test]
    fn rejects_wrapper_properties_without_an_executable() {
        let directory = tempfile::tempdir().unwrap();
        fs::create_dir_all(directory.path().join(".mvn/wrapper")).unwrap();
        fs::write(
            directory
                .path()
                .join(".mvn/wrapper/maven-wrapper.properties"),
            "distributionUrl=https://example.test/apache-maven-3.9.9-bin.zip\n",
        )
        .unwrap();
        let java = JdkInstallation::recorded(17, directory.path().join("jdk-17"));

        assert!(matches!(
            detect_maven(directory.path(), &java, &mut |_| {}),
            Err(ProjectDetectionError::MavenWrapperNotFound { .. })
        ));
    }

    #[test]
    fn rejects_non_maven_projects_with_a_typed_error() {
        let directory = tempfile::tempdir().unwrap();
        let error = ProjectEnvironment::detect(directory.path()).unwrap_err();
        assert!(matches!(
            error,
            ProjectDetectionError::NotMavenProject { .. }
        ));
    }
}
