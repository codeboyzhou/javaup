//! Project environment detection and persisted project configuration.

use std::path::{Path, PathBuf};

use crate::java::{JdkInstallation, JdkValidationError};

mod build;
mod config;
mod detection;

pub use build::ProjectBuildError;
pub use config::ProjectConfigError;
pub use detection::ProjectDetectionError;

/// A meaningful milestone emitted while detecting a project environment.
///
/// Consumers can use these events to provide progress feedback without
/// coupling the core detection logic to a particular user interface.
#[derive(Clone, Debug, Eq, PartialEq)]
#[non_exhaustive]
pub enum ProjectDetectionEvent {
    InspectingProject { project_dir: PathBuf },
    ReadingJavaRequirements { pom_path: PathBuf },
    SearchingForJdk { major_version: u32 },
    JdkDetected { major_version: u32, home: PathBuf },
    ReadingMavenWrapper { properties_path: PathBuf },
    MavenWrapperUnavailable,
    MavenDetected { version: String, uses_wrapper: bool },
}

/// Build system used by a detected project.
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
#[non_exhaustive]
pub enum ProjectType {
    Maven,
}

impl ProjectType {
    pub fn as_str(self) -> &'static str {
        match self {
            Self::Maven => "maven",
        }
    }
}

/// Maven installation required to build a project.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct MavenEnvironment {
    version: String,
    uses_wrapper: bool,
}

impl MavenEnvironment {
    pub fn version(&self) -> &str {
        &self.version
    }

    pub fn uses_wrapper(&self) -> bool {
        self.uses_wrapper
    }
}

/// Reproducible build environment detected for a project.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct ProjectEnvironment {
    project_type: ProjectType,
    java: JdkInstallation,
    maven: MavenEnvironment,
}

impl ProjectEnvironment {
    /// Detects the build environment required by the project at `project_dir`.
    pub fn detect(project_dir: impl AsRef<Path>) -> Result<Self, ProjectDetectionError> {
        Self::detect_with_observer(project_dir, |_| {})
    }

    /// Detects the project environment and reports meaningful milestones.
    ///
    /// The observer is deliberately best-effort: it cannot alter detection or
    /// replace a domain error with a presentation-layer error.
    pub fn detect_with_observer<F>(
        project_dir: impl AsRef<Path>,
        mut observer: F,
    ) -> Result<Self, ProjectDetectionError>
    where
        F: FnMut(ProjectDetectionEvent),
    {
        detection::detect(project_dir.as_ref(), &mut observer)
    }

    /// Loads a previously saved environment for `project_dir`.
    pub fn load(project_dir: impl AsRef<Path>) -> Result<Self, ProjectConfigError> {
        config::load(project_dir.as_ref())
    }

    /// Loads the nearest saved environment from `start` or one of its parent directories.
    pub fn load_nearest(start: impl AsRef<Path>) -> Result<(PathBuf, Self), ProjectConfigError> {
        config::load_nearest(start.as_ref())
    }

    /// Returns the user-level configuration path for `project_dir`.
    pub fn configuration_path(
        project_dir: impl AsRef<Path>,
    ) -> Result<PathBuf, ProjectConfigError> {
        config::configuration_path(project_dir.as_ref())
    }

    /// Saves this environment in javaup's user-level configuration directory.
    pub fn save(&self, project_dir: impl AsRef<Path>) -> Result<PathBuf, ProjectConfigError> {
        config::save(project_dir.as_ref(), self)
    }

    pub fn project_type(&self) -> ProjectType {
        self.project_type
    }

    pub fn java_version(&self) -> u32 {
        self.java.major_version()
    }

    /// Re-validates the recorded JDK and returns its full installed version.
    pub fn installed_java_version(&self) -> Result<String, JdkValidationError> {
        self.java.version()
    }

    pub fn java_home(&self) -> &Path {
        self.java.home()
    }

    pub fn maven(&self) -> &MavenEnvironment {
        &self.maven
    }

    /// Creates a Maven command configured to use this environment's JDK.
    pub fn maven_command<I, S>(
        &self,
        project_dir: impl AsRef<Path>,
        arguments: I,
    ) -> Result<std::process::Command, ProjectBuildError>
    where
        I: IntoIterator<Item = S>,
        S: AsRef<std::ffi::OsStr>,
    {
        build::maven_command(project_dir.as_ref(), self, arguments)
    }
}

fn is_maven_version(value: &str) -> bool {
    !value.is_empty()
        && value.chars().next().is_some_and(|c| c.is_ascii_digit())
        && value
            .chars()
            .all(|c| c.is_ascii_alphanumeric() || matches!(c, '.' | '-' | '_' | '+'))
}
