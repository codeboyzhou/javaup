use std::env;
use std::error::Error;
use std::ffi::OsStr;
use std::fmt;
use std::path::{Path, PathBuf};
use std::process::Command;

use super::ProjectEnvironment;
use crate::java::JdkValidationError;

/// Error raised while preparing a project build command.
#[derive(Debug)]
#[non_exhaustive]
pub enum ProjectBuildError {
    InvalidJdk { source: JdkValidationError },
    MavenWrapperNotFound { project_dir: PathBuf },
    MavenCommandUnavailable,
    InvalidPath { source: env::JoinPathsError },
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
            Self::InvalidPath { source } => Some(source),
            _ => None,
        }
    }
}

pub(super) fn maven_command<I, S>(
    project_dir: &Path,
    environment: &ProjectEnvironment,
    arguments: I,
) -> Result<Command, ProjectBuildError>
where
    I: IntoIterator<Item = S>,
    S: AsRef<OsStr>,
{
    environment
        .java
        .validate()
        .map_err(|source| ProjectBuildError::InvalidJdk { source })?;

    let executable = if environment.maven.uses_wrapper {
        find_wrapper(project_dir).ok_or_else(|| ProjectBuildError::MavenWrapperNotFound {
            project_dir: project_dir.to_owned(),
        })?
    } else {
        find_maven_on_path().ok_or(ProjectBuildError::MavenCommandUnavailable)?
    };

    let mut paths = vec![environment.java.home().join("bin")];
    if let Some(path) = env::var_os("PATH") {
        paths.extend(env::split_paths(&path));
    }
    let path =
        env::join_paths(paths).map_err(|source| ProjectBuildError::InvalidPath { source })?;

    let mut command = Command::new(executable);
    command
        .args(arguments)
        .current_dir(project_dir)
        .env("JAVA_HOME", environment.java.home())
        .env("PATH", path);
    Ok(command)
}

fn find_wrapper(project_dir: &Path) -> Option<PathBuf> {
    wrapper_names()
        .iter()
        .map(|name| project_dir.join(name))
        .find(|path| path.is_file())
}

fn find_maven_on_path() -> Option<PathBuf> {
    let path = env::var_os("PATH")?;
    for directory in env::split_paths(&path) {
        for name in maven_command_names() {
            let candidate = directory.join(name);
            if candidate.is_file() {
                return Some(candidate);
            }
        }
    }
    None
}

#[cfg(windows)]
fn wrapper_names() -> &'static [&'static str] {
    &["mvnw.cmd", "mvnw.bat", "mvnw.exe", "mvnw"]
}

#[cfg(not(windows))]
fn wrapper_names() -> &'static [&'static str] {
    &["mvnw"]
}

#[cfg(windows)]
fn maven_command_names() -> &'static [&'static str] {
    &["mvn.cmd", "mvn.bat", "mvn.exe", "mvn"]
}

#[cfg(not(windows))]
fn maven_command_names() -> &'static [&'static str] {
    &["mvn"]
}

#[cfg(test)]
mod tests {
    use std::fs;

    use super::*;
    use crate::java::JdkInstallation;
    use crate::project::{MavenEnvironment, ProjectType};

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

        let wrapper = directory.path().join(wrapper_names()[0]);
        fs::write(&wrapper, "").unwrap();
        let environment = ProjectEnvironment {
            project_type: ProjectType::Maven,
            java: JdkInstallation::recorded(17, java_home.clone()),
            maven: MavenEnvironment {
                version: "3.9.9".to_owned(),
                uses_wrapper: true,
                settings_profile: None,
            },
        };

        let command = maven_command(directory.path(), &environment, ["clean", "package"]).unwrap();

        assert_eq!(command.get_program(), wrapper.as_os_str());
        assert_eq!(command.get_current_dir(), Some(directory.path()));
        assert_eq!(
            command
                .get_envs()
                .find(|(key, _)| *key == OsStr::new("JAVA_HOME"))
                .and_then(|(_, value)| value),
            Some(java_home.as_os_str())
        );
        assert_eq!(
            command.get_args().collect::<Vec<_>>(),
            vec![OsStr::new("clean"), OsStr::new("package")]
        );
    }
}
