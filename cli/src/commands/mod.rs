use std::error::Error;
use std::fmt;
use std::io::{self, Write};
use std::process::ExitStatus;

use javaup_core::java::JdkValidationError;
use javaup_core::maven_settings::MavenSettingsError;
use javaup_core::project::{ProjectBuildError, ProjectConfigError, ProjectDetectionError};

use crate::cli::Command;
use crate::output::Output;

mod init;
mod mvn;
mod settings;
mod status;
mod version;

pub(crate) enum CommandOutcome {
    Success,
    Process(ExitStatus),
}

#[derive(Debug)]
pub(crate) enum CommandError {
    Io(io::Error),
    ProjectDetection(ProjectDetectionError),
    ProjectConfig(ProjectConfigError),
    ProjectBuild(ProjectBuildError),
    MavenSettings(MavenSettingsError),
    JdkValidation(JdkValidationError),
    UnsupportedBuildTool {
        expected: &'static str,
        actual: &'static str,
    },
}

impl fmt::Display for CommandError {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Io(error) => error.fmt(formatter),
            Self::ProjectDetection(error) => error.fmt(formatter),
            Self::ProjectConfig(error) => error.fmt(formatter),
            Self::ProjectBuild(error) => error.fmt(formatter),
            Self::MavenSettings(error) => error.fmt(formatter),
            Self::JdkValidation(error) => error.fmt(formatter),
            Self::UnsupportedBuildTool { expected, actual } => write!(
                formatter,
                "this command requires a {expected} project, but the initialized project uses {actual}"
            ),
        }
    }
}

impl Error for CommandError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::Io(error) => Some(error),
            Self::ProjectDetection(error) => Some(error),
            Self::ProjectConfig(error) => Some(error),
            Self::ProjectBuild(error) => Some(error),
            Self::MavenSettings(error) => Some(error),
            Self::JdkValidation(error) => Some(error),
            Self::UnsupportedBuildTool { .. } => None,
        }
    }
}

impl From<io::Error> for CommandError {
    fn from(error: io::Error) -> Self {
        Self::Io(error)
    }
}

impl From<ProjectDetectionError> for CommandError {
    fn from(error: ProjectDetectionError) -> Self {
        Self::ProjectDetection(error)
    }
}

impl From<ProjectConfigError> for CommandError {
    fn from(error: ProjectConfigError) -> Self {
        Self::ProjectConfig(error)
    }
}

impl From<ProjectBuildError> for CommandError {
    fn from(error: ProjectBuildError) -> Self {
        Self::ProjectBuild(error)
    }
}

impl From<MavenSettingsError> for CommandError {
    fn from(error: MavenSettingsError) -> Self {
        Self::MavenSettings(error)
    }
}

impl From<JdkValidationError> for CommandError {
    fn from(error: JdkValidationError) -> Self {
        Self::JdkValidation(error)
    }
}

impl CommandOutcome {
    pub(crate) fn exit_code(&self) -> u8 {
        match self {
            Self::Success => 0,
            Self::Process(status) if status.success() => 0,
            Self::Process(status) => status
                .code()
                .and_then(|code| u8::try_from(code).ok())
                .unwrap_or(1),
        }
    }
}

pub(crate) fn execute<Stdout, Stderr>(
    command: Command,
    output: &mut Output<'_, Stdout, Stderr>,
) -> Result<CommandOutcome, CommandError>
where
    Stdout: Write,
    Stderr: Write,
{
    match command {
        Command::Init => init::execute(output).map(|_| CommandOutcome::Success),
        Command::Mvn(arguments) => {
            mvn::execute(arguments.maven_arguments, output).map(CommandOutcome::Process)
        }
        Command::Settings(arguments) => {
            settings::execute(arguments.command, output).map(|_| CommandOutcome::Success)
        }
        Command::Status => status::execute(output).map(|_| CommandOutcome::Success),
        Command::Version => {
            version::execute(output.stdout())?;
            Ok(CommandOutcome::Success)
        }
    }
}
