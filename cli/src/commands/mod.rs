use std::io::{self, Write};

use crate::cli::Command;

mod version;

pub(crate) fn execute<W>(command: Command, stdout: &mut W) -> io::Result<()>
where
    W: Write,
{
    match command {
        Command::Version => version::execute(stdout),
    }
}
