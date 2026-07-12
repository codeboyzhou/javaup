//! Project environment detection and persisted project configuration.

use std::path::{Path, PathBuf};

mod config;
mod detection;

pub use config::ProjectConfigError;
pub use detection::ProjectDetectionError;

/// Name of the environment file stored in a project's root directory.
pub const ENVIRONMENT_FILE_NAME: &str = ".javaup";

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
    java_version: u32,
    maven: MavenEnvironment,
}

impl ProjectEnvironment {
    /// Detects the build environment required by the project at `project_dir`.
    pub fn detect(project_dir: impl AsRef<Path>) -> Result<Self, ProjectDetectionError> {
        detection::detect(project_dir.as_ref())
    }

    /// Loads a previously saved environment from `.javaup`.
    pub fn load(project_dir: impl AsRef<Path>) -> Result<Self, ProjectConfigError> {
        config::load(project_dir.as_ref())
    }

    /// Saves this environment to `.javaup` and returns the written path.
    pub fn save(&self, project_dir: impl AsRef<Path>) -> Result<PathBuf, ProjectConfigError> {
        config::save(project_dir.as_ref(), self)
    }

    pub fn project_type(&self) -> ProjectType {
        self.project_type
    }

    pub fn java_version(&self) -> u32 {
        self.java_version
    }

    pub fn maven(&self) -> &MavenEnvironment {
        &self.maven
    }
}

fn is_maven_version(value: &str) -> bool {
    !value.is_empty()
        && value.chars().next().is_some_and(|c| c.is_ascii_digit())
        && value
            .chars()
            .all(|c| c.is_ascii_alphanumeric() || matches!(c, '.' | '-' | '_' | '+'))
}
