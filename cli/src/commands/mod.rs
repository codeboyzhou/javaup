use std::io::{self, Write};

use crate::cli::Command;

mod init;
mod version;

pub(crate) fn execute<W>(command: Command, stdout: &mut W) -> io::Result<()>
where
    W: Write,
{
    match command {
        Command::Init => init::execute(stdout),
        Command::Version => version::execute(stdout),
    }
}
