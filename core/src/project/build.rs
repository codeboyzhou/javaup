use std::error::Error;
use std::ffi::OsStr;
use std::fmt;
use std::io;
use std::path::{Path, PathBuf};
use std::process::ExitStatus;

use super::{ProjectEnvironment, maven};
use crate::java::JdkValidationError;
use crate::process::{ProcessInvocation, ProcessRunner};

/// Error raised while preparing a project build command.
#[derive(Debug)]
#[non_exhaustive]
pub enum ProjectBuildError {
    InvalidJdk {
        source: JdkValidationError,
    },
    MavenWrapperNotFound {
        project_dir: PathBuf,
    },
    MavenCommandUnavailable,
    UnsupportedBuildTool {
        actual: &'static str,
    },
    MavenConfigurationRead {
        path: PathBuf,
        source: io::Error,
    },
    MavenVersionCommand {
        path: PathBuf,
        source: io::Error,
    },
    MavenVersionCommandFailed {
        path: PathBuf,
        status: ExitStatus,
    },
    MavenVersionNotFound {
        path: PathBuf,
    },
    MavenVersionMismatch {
        path: PathBuf,
        expected: String,
        actual: String,
    },
    InvalidPath {
        source: std::env::JoinPathsError,
    },
}

impl fmt::Display for ProjectBuildError {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::InvalidJdk { source } => {
                write!(formatter, "recorded JDK is not usable: {source}")
            }
            Self::MavenWrapperNotFound { project_dir } => write!(
                formatter,
                "Maven Wrapper is enabled, but no wrapper executable was found in {}",
                project_dir.display()
            ),
            Self::MavenCommandUnavailable => {
                write!(formatter, "Maven is not available on PATH")
            }
            Self::UnsupportedBuildTool { actual } => {
                write!(
                    formatter,
                    "cannot create a Maven invocation for a {actual} project"
                )
            }
            Self::MavenConfigurationRead { path, source } => write!(
                formatter,
                "could not read Maven configuration {}: {source}",
                path.display()
            ),
            Self::MavenVersionCommand { path, source } => write!(
                formatter,
                "could not run '{} --version': {source}",
                path.display()
            ),
            Self::MavenVersionCommandFailed { path, status } => write!(
                formatter,
                "'{} --version' failed with status {status}",
                path.display()
            ),
            Self::MavenVersionNotFound { path } => write!(
                formatter,
                "could not determine the Maven version for {}",
                path.display()
            ),
            Self::MavenVersionMismatch {
                path,
                expected,
                actual,
            } => write!(
                formatter,
                "{} provides Maven {actual}, but the project was initialized with Maven {expected}; run 'jup init' again or restore the recorded Maven version",
                path.display()
            ),
            Self::InvalidPath { source } => {
                write!(
                    formatter,
                    "could not construct PATH for the selected JDK: {source}"
                )
            }
        }
    }
}

impl Error for ProjectBuildError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::InvalidJdk { source } => Some(source),
            Self::MavenConfigurationRead { source, .. }
            | Self::MavenVersionCommand { source, .. } => Some(source),
            Self::InvalidPath { source } => Some(source),
            _ => None,
        }
    }
}

pub(super) fn maven_invocation<I, S, R>(
    project_dir: &Path,
    environment: &ProjectEnvironment,
    arguments: I,
    runner: &R,
) -> Result<ProcessInvocation, ProjectBuildError>
where
    I: IntoIterator<Item = S>,
    S: AsRef<OsStr>,
    R: ProcessRunner + ?Sized,
{
    environment
        .java
        .validate()
        .map_err(|source| ProjectBuildError::InvalidJdk { source })?;

    let executable = validate_maven(project_dir, environment, runner)?;
    let arguments = arguments
        .into_iter()
        .map(|argument| argument.as_ref().to_owned())
        .collect::<Vec<_>>();
    maven::invocation(&executable, project_dir, environment.java.home(), arguments)
        .map_err(|source| ProjectBuildError::InvalidPath { source })
}

fn validate_maven(
    project_dir: &Path,
    environment: &ProjectEnvironment,
    runner: &(impl ProcessRunner + ?Sized),
) -> Result<PathBuf, ProjectBuildError> {
    let maven_environment = environment
        .maven()
        .ok_or(ProjectBuildError::UnsupportedBuildTool {
            actual: environment.build_tool().as_str(),
        })?;
    if maven_environment.uses_wrapper {
        let executable = maven::find_wrapper(project_dir).ok_or_else(|| {
            ProjectBuildError::MavenWrapperNotFound {
                project_dir: project_dir.to_owned(),
            }
        })?;
        let properties_path = maven::wrapper_properties_path(project_dir);
        let actual = maven::read_wrapper_version(&properties_path)
            .map_err(|source| ProjectBuildError::MavenConfigurationRead {
                path: properties_path.clone(),
                source,
            })?
            .ok_or(ProjectBuildError::MavenVersionNotFound {
                path: properties_path,
            })?;
        ensure_version_matches(&executable, &maven_environment.version, actual)?;
        return Ok(executable);
    }

    let executable =
        maven::find_maven_on_path().ok_or(ProjectBuildError::MavenCommandUnavailable)?;
    let output =
        maven::run_maven_version(runner, &executable, project_dir, environment.java.home())
            .map_err(|error| match error {
                maven::MavenProbeError::Command(source) => ProjectBuildError::MavenVersionCommand {
                    path: executable.clone(),
                    source,
                },
                maven::MavenProbeError::InvalidPath(source) => {
                    ProjectBuildError::InvalidPath { source }
                }
            })?;
    if !output.status().success() {
        return Err(ProjectBuildError::MavenVersionCommandFailed {
            path: executable,
            status: output.status(),
        });
    }
    let combined = format!(
        "{}\n{}",
        String::from_utf8_lossy(output.stdout()),
        String::from_utf8_lossy(output.stderr())
    );
    let actual = maven::parse_maven_version_output(&combined).ok_or_else(|| {
        ProjectBuildError::MavenVersionNotFound {
            path: executable.clone(),
        }
    })?;
    ensure_version_matches(&executable, &maven_environment.version, actual)?;
    Ok(executable)
}

fn ensure_version_matches(
    executable: &Path,
    expected: &str,
    actual: String,
) -> Result<(), ProjectBuildError> {
    if actual == expected {
        return Ok(());
    }
    Err(ProjectBuildError::MavenVersionMismatch {
        path: executable.to_owned(),
        expected: expected.to_owned(),
        actual,
    })
}

#[cfg(test)]
mod tests {
    use std::fs;

    use super::*;
    use crate::java::JdkInstallation;
    use crate::process::SystemProcessRunner;
    use crate::project::{BuildToolEnvironment, MavenEnvironment};

    #[test]
    fn prepares_a_wrapper_command_with_the_recorded_jdk() {
        let directory = tempfile::tempdir().unwrap();
        let java_home = directory.path().join("jdk-17");
        fs::create_dir(&java_home).unwrap();
        fs::create_dir(java_home.join("bin")).unwrap();
        fs::write(java_home.join("release"), "JAVA_VERSION=\"17.0.12\"\n").unwrap();
        fs::write(
            java_home
                .join("bin")
                .join(if cfg!(windows) { "java.exe" } else { "java" }),
            "",
        )
        .unwrap();
        fs::write(
            java_home
                .join("bin")
                .join(if cfg!(windows) { "javac.exe" } else { "javac" }),
            "",
        )
        .unwrap();
        #[cfg(unix)]
        for executable in ["java", "javac"] {
            use std::os::unix::fs::PermissionsExt;

            fs::set_permissions(
                java_home.join("bin").join(executable),
                fs::Permissions::from_mode(0o755),
            )
            .unwrap();
        }

        let wrapper = directory
            .path()
            .join(if cfg!(windows) { "mvnw.cmd" } else { "mvnw" });
        fs::write(&wrapper, "").unwrap();
        #[cfg(unix)]
        {
            use std::os::unix::fs::PermissionsExt;
            fs::set_permissions(&wrapper, fs::Permissions::from_mode(0o755)).unwrap();
        }
        fs::create_dir_all(directory.path().join(".mvn/wrapper")).unwrap();
        fs::write(
            directory
                .path()
                .join(".mvn/wrapper/maven-wrapper.properties"),
            "distributionUrl=https://example.test/apache-maven-3.9.9-bin.zip\n",
        )
        .unwrap();
        let environment = ProjectEnvironment {
            java: JdkInstallation::recorded(17, java_home.clone()),
            build_tool: BuildToolEnvironment::Maven(MavenEnvironment {
                version: "3.9.9".to_owned(),
                uses_wrapper: true,
                settings_profile: None,
            }),
        };

        let invocation = maven_invocation(
            directory.path(),
            &environment,
            ["clean", "package"],
            &SystemProcessRunner,
        )
        .unwrap();

        assert_eq!(invocation.program(), wrapper.as_path());
        assert_eq!(invocation.current_dir(), directory.path());
        assert_eq!(
            invocation
                .environment()
                .find(|(key, _)| *key == OsStr::new("JAVA_HOME"))
                .map(|(_, value)| value),
            Some(java_home.as_os_str())
        );
        assert_eq!(
            invocation.arguments().collect::<Vec<_>>(),
            vec![OsStr::new("clean"), OsStr::new("package")]
        );
    }

    #[test]
    fn rejects_a_maven_version_that_changed_after_initialization() {
        let executable = Path::new("mvn");
        let error =
            ensure_version_matches(executable, "3.9.9", "4.0.0-rc-4".to_owned()).unwrap_err();

        assert!(matches!(
            error,
            ProjectBuildError::MavenVersionMismatch {
                expected,
                actual,
                ..
            } if expected == "3.9.9" && actual == "4.0.0-rc-4"
        ));
    }
}
