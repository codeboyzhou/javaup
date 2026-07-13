use std::io::{self, Write};
use std::process::ExitStatus;

use crate::cli::Command;

mod build;
mod init;
mod version;

pub(crate) enum CommandOutcome {
    Success,
    Process(ExitStatus),
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

pub(crate) fn execute<W>(command: Command, stdout: &mut W) -> io::Result<CommandOutcome>
where
    W: Write,
{
    match command {
        Command::Init => init::execute(stdout).map(|_| CommandOutcome::Success),
        Command::Build(arguments) => {
            build::execute(arguments.maven_arguments, stdout).map(CommandOutcome::Process)
        }
        Command::Version => version::execute(stdout).map(|_| CommandOutcome::Success),
    }
}
