use std::collections::{HashMap, HashSet};
use std::error::Error;
use std::fmt;
use std::fs;
use std::io;
use std::path::{Path, PathBuf};
use std::process::{Command, ExitStatus, Output};

use super::{MavenEnvironment, ProjectEnvironment, ProjectType, is_maven_version};

const POM_FILE_NAME: &str = "pom.xml";

/// Error raised while detecting a project's build environment.
#[derive(Debug)]
#[non_exhaustive]
pub enum ProjectDetectionError {
    NotMavenProject { project_dir: PathBuf },
    FileAccess { path: PathBuf, source: io::Error },
    InvalidPom { path: PathBuf, message: String },
    JavaVersionNotFound { pom_path: PathBuf },
    MavenWrapperVersionNotFound { properties_path: PathBuf },
    MavenCommandUnavailable { source: io::Error },
    MavenCommandFailed { status: ExitStatus },
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
            Self::MavenWrapperVersionNotFound { properties_path } => write!(
                formatter,
                "could not determine the Maven version from {}",
                properties_path.display()
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
            _ => None,
        }
    }
}

pub(super) fn detect(project_dir: &Path) -> Result<ProjectEnvironment, ProjectDetectionError> {
    let pom_path = project_dir.join(POM_FILE_NAME);
    if !pom_path.is_file() {
        return Err(ProjectDetectionError::NotMavenProject {
            project_dir: project_dir.to_owned(),
        });
    }

    let java_version = detect_java_version(&pom_path)?;
    let maven = detect_maven(project_dir)?;
    Ok(ProjectEnvironment {
        project_type: ProjectType::Maven,
        java_version,
        maven,
    })
}

fn detect_maven(project_dir: &Path) -> Result<MavenEnvironment, ProjectDetectionError> {
    let wrapper_properties = project_dir.join(".mvn/wrapper/maven-wrapper.properties");
    if wrapper_properties.is_file() {
        let contents = read_to_string(&wrapper_properties)?;
        let version = parse_wrapper_maven_version(&contents).ok_or({
            ProjectDetectionError::MavenWrapperVersionNotFound {
                properties_path: wrapper_properties,
            }
        })?;
        return Ok(MavenEnvironment {
            version,
            uses_wrapper: true,
        });
    }

    let output = run_maven_version(project_dir)
        .map_err(|source| ProjectDetectionError::MavenCommandUnavailable { source })?;
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
    let version =
        parse_maven_version_output(&combined).ok_or(ProjectDetectionError::MavenVersionNotFound)?;
    Ok(MavenEnvironment {
        version,
        uses_wrapper: false,
    })
}

fn run_maven_version(project_dir: &Path) -> io::Result<Output> {
    let mut last_not_found = None;
    for executable in maven_command_candidates() {
        match Command::new(executable)
            .arg("--version")
            .current_dir(project_dir)
            .output()
        {
            Ok(output) => return Ok(output),
            Err(error) if error.kind() == io::ErrorKind::NotFound => {
                last_not_found = Some(error);
            }
            Err(error) => return Err(error),
        }
    }

    Err(last_not_found.expect("Maven command candidates must not be empty"))
}

#[cfg(windows)]
fn maven_command_candidates() -> &'static [&'static str] {
    &["mvn.cmd", "mvn.bat", "mvn.exe", "mvn"]
}

#[cfg(not(windows))]
fn maven_command_candidates() -> &'static [&'static str] {
    &["mvn"]
}

fn parse_wrapper_maven_version(contents: &str) -> Option<String> {
    let distribution_url = contents.lines().find_map(|line| {
        let line = line.trim();
        if line.starts_with('#') || line.starts_with('!') {
            return None;
        }
        let (key, value) = line.split_once('=')?;
        (key.trim() == "distributionUrl").then(|| value.trim().replace("\\:", ":"))
    })?;

    extract_version_after_marker(&distribution_url, "apache-maven-")
        .or_else(|| extract_version_after_marker(&distribution_url, "maven-mvnd-"))
}

fn extract_version_after_marker(value: &str, marker: &str) -> Option<String> {
    let remainder = value.rsplit_once(marker)?.1;
    let archive_name = remainder.split(['?', '#']).next()?.trim();
    let version = ["-bin.zip", "-bin.tar.gz", ".zip", ".tar.gz"]
        .into_iter()
        .find_map(|suffix| archive_name.strip_suffix(suffix))?;
    is_maven_version(version).then(|| version.to_owned())
}

fn parse_maven_version_output(output: &str) -> Option<String> {
    output.lines().find_map(|line| {
        let line = strip_ansi_codes(line);
        let remainder = line.trim().strip_prefix("Apache Maven ")?;
        let version = remainder.split_whitespace().next()?;
        is_maven_version(version).then(|| version.to_owned())
    })
}

fn strip_ansi_codes(value: &str) -> String {
    let mut result = String::with_capacity(value.len());
    let mut chars = value.chars().peekable();
    while let Some(character) = chars.next() {
        if character == '\u{1b}' && chars.peek() == Some(&'[') {
            chars.next();
            for sequence_character in chars.by_ref() {
                if sequence_character.is_ascii_alphabetic() {
                    break;
                }
            }
        } else {
            result.push(character);
        }
    }
    result
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
        let relative_path = child(parent, "relativePath")
            .and_then(text)
            .unwrap_or_else(|| "../pom.xml".to_owned());
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
    for plugin in project
        .descendants()
        .filter(|node| node.is_element() && node.tag_name().name() == "plugin")
    {
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

fn parse_java_major(value: &str) -> Option<u32> {
    let value = value.trim().trim_start_matches(['[', '(']);
    let major = value.strip_prefix("1.").unwrap_or(value);
    let digits: String = major
        .chars()
        .take_while(|character| character.is_ascii_digit())
        .collect();
    let major = digits.parse().ok()?;
    (major >= 5).then_some(major)
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
            parse_wrapper_maven_version(
                "distributionUrl=https\\://repo.maven.apache.org/maven2/org/apache/maven/apache-maven/3.9.9/apache-maven-3.9.9-bin.zip"
            ),
            Some("3.9.9".to_owned())
        );
        assert_eq!(
            parse_wrapper_maven_version(
                "distributionUrl=https://example.test/apache-maven-4.0.0-rc-4-bin.zip"
            ),
            Some("4.0.0-rc-4".to_owned())
        );
        assert_eq!(
            parse_maven_version_output("Apache Maven 3.9.11 (abcdef)\nMaven home: /opt/maven"),
            Some("3.9.11".to_owned())
        );
    }

    #[cfg(windows)]
    #[test]
    fn prefers_the_windows_maven_command_launcher() {
        assert_eq!(maven_command_candidates().first(), Some(&"mvn.cmd"));
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
    fn detects_a_maven_wrapper_project() {
        let directory = tempfile::tempdir().unwrap();
        fs::create_dir_all(directory.path().join(".mvn/wrapper")).unwrap();
        fs::write(
            directory.path().join(POM_FILE_NAME),
            r#"<project><properties><java.version>1.8</java.version></properties></project>"#,
        )
            .unwrap();
        fs::write(
            directory
                .path()
                .join(".mvn/wrapper/maven-wrapper.properties"),
            "distributionUrl=https://example.test/apache-maven-3.9.9-bin.zip\n",
        )
            .unwrap();

        let environment = ProjectEnvironment::detect(directory.path()).unwrap();

        assert_eq!(environment.project_type(), ProjectType::Maven);
        assert_eq!(environment.java_version(), 8);
        assert_eq!(environment.maven().version(), "3.9.9");
        assert!(environment.maven().uses_wrapper());
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
