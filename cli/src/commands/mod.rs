use std::io::{self, Write};
use std::process::ExitStatus;

use crate::cli::Command;
use crate::output::Output;

mod build;
mod init;
mod status;
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

pub(crate) fn execute<Stdout, Stderr>(
    command: Command,
    output: &mut Output<'_, Stdout, Stderr>,
) -> io::Result<CommandOutcome>
where
    Stdout: Write,
    Stderr: Write,
{
    match command {
        Command::Init => init::execute(output).map(|_| CommandOutcome::Success),
        Command::Build(arguments) => {
            build::execute(arguments.maven_arguments, output.stdout()).map(CommandOutcome::Process)
        }
        Command::Status => status::execute(output).map(|_| CommandOutcome::Success),
        Command::Version => version::execute(output.stdout()).map(|_| CommandOutcome::Success),
    }
}
