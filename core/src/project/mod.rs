//! Project environment detection and persisted project configuration.

use std::path::{Path, PathBuf};

use crate::java::{JdkInstallation, JdkValidationError};
use crate::maven_settings::MavenSettingsProfile;
use crate::process::{ProcessInvocation, ProcessRunner, SystemProcessRunner};

mod build;
mod config;
mod detection;
mod maven;

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
    settings_profile: Option<String>,
}

impl MavenEnvironment {
    pub fn version(&self) -> &str {
        &self.version
    }

    pub fn uses_wrapper(&self) -> bool {
        self.uses_wrapper
    }

    /// Returns the named Maven settings profile bound to this project.
    pub fn settings_profile(&self) -> Option<&str> {
        self.settings_profile.as_deref()
    }
}

/// Build-tool-specific state required to reproduce a project build.
#[derive(Clone, Debug, Eq, PartialEq)]
#[non_exhaustive]
pub enum BuildToolEnvironment {
    Maven(MavenEnvironment),
}

impl BuildToolEnvironment {
    pub fn project_type(&self) -> ProjectType {
        match self {
            Self::Maven(_) => ProjectType::Maven,
        }
    }

    pub fn as_str(&self) -> &'static str {
        self.project_type().as_str()
    }

    pub fn as_maven(&self) -> Option<&MavenEnvironment> {
        match self {
            Self::Maven(environment) => Some(environment),
        }
    }

    fn as_maven_mut(&mut self) -> Option<&mut MavenEnvironment> {
        match self {
            Self::Maven(environment) => Some(environment),
        }
    }
}

/// Reproducible build environment detected for a project.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct ProjectEnvironment {
    java: JdkInstallation,
    build_tool: BuildToolEnvironment,
}

impl ProjectEnvironment {
    /// Detects the build environment required by the project at `project_dir`.
    pub fn detect(project_dir: impl AsRef<Path>) -> Result<Self, ProjectDetectionError> {
        Self::detect_with_runner_and_observer(project_dir, &SystemProcessRunner, |_| {})
    }

    /// Detects the project environment and reports meaningful milestones.
    ///
    /// The observer is deliberately best-effort: it cannot alter detection or
    /// replace a domain error with a presentation-layer error.
    pub fn detect_with_observer<F>(
        project_dir: impl AsRef<Path>,
        observer: F,
    ) -> Result<Self, ProjectDetectionError>
    where
        F: FnMut(ProjectDetectionEvent),
    {
        Self::detect_with_runner_and_observer(project_dir, &SystemProcessRunner, observer)
    }

    /// Detects a project using a caller-provided process runner.
    pub fn detect_with_runner<R>(
        project_dir: impl AsRef<Path>,
        runner: &R,
    ) -> Result<Self, ProjectDetectionError>
    where
        R: ProcessRunner + ?Sized,
    {
        Self::detect_with_runner_and_observer(project_dir, runner, |_| {})
    }

    /// Detects a project using caller-provided process and progress adapters.
    pub fn detect_with_runner_and_observer<R, F>(
        project_dir: impl AsRef<Path>,
        runner: &R,
        mut observer: F,
    ) -> Result<Self, ProjectDetectionError>
    where
        R: ProcessRunner + ?Sized,
        F: FnMut(ProjectDetectionEvent),
    {
        detection::detect(project_dir.as_ref(), runner, &mut observer)
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

    /// Saves this environment while retaining an existing Maven settings binding.
    pub fn save_preserving_maven_settings(
        &self,
        project_dir: impl AsRef<Path>,
    ) -> Result<PathBuf, ProjectConfigError> {
        config::save_preserving_maven_settings(project_dir.as_ref(), self)
    }

    /// Binds or clears a registered Maven settings profile for this project.
    pub fn set_maven_settings(&mut self, profile: Option<&MavenSettingsProfile>) -> bool {
        let Some(maven) = self.build_tool.as_maven_mut() else {
            return false;
        };
        maven.settings_profile = profile.map(|profile| profile.name().to_owned());
        true
    }

    pub fn project_type(&self) -> ProjectType {
        self.build_tool.project_type()
    }

    pub fn build_tool(&self) -> &BuildToolEnvironment {
        &self.build_tool
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

    pub fn maven(&self) -> Option<&MavenEnvironment> {
        self.build_tool.as_maven()
    }

    /// Describes a Maven process configured to use this environment's JDK.
    pub fn maven_invocation<I, S>(
        &self,
        project_dir: impl AsRef<Path>,
        arguments: I,
    ) -> Result<ProcessInvocation, ProjectBuildError>
    where
        I: IntoIterator<Item = S>,
        S: AsRef<std::ffi::OsStr>,
    {
        self.maven_invocation_with_runner(project_dir, arguments, &SystemProcessRunner)
    }

    /// Describes a Maven process after validating it with a custom runner.
    pub fn maven_invocation_with_runner<I, S, R>(
        &self,
        project_dir: impl AsRef<Path>,
        arguments: I,
        runner: &R,
    ) -> Result<ProcessInvocation, ProjectBuildError>
    where
        I: IntoIterator<Item = S>,
        S: AsRef<std::ffi::OsStr>,
        R: ProcessRunner + ?Sized,
    {
        build::maven_invocation(project_dir.as_ref(), self, arguments, runner)
    }
}

fn is_maven_version(value: &str) -> bool {
    !value.is_empty()
        && value.chars().next().is_some_and(|c| c.is_ascii_digit())
        && value
            .chars()
            .all(|c| c.is_ascii_alphanumeric() || matches!(c, '.' | '-' | '_' | '+'))
}
