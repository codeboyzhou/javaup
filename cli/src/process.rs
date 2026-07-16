use std::io;
use std::process::{Command, ExitStatus};

use javaup_core::process::ProcessInvocation;

pub(crate) fn status(invocation: &ProcessInvocation) -> io::Result<ExitStatus> {
    Command::new(invocation.program())
        .args(invocation.arguments())
        .current_dir(invocation.current_dir())
        .envs(invocation.environment())
        .status()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn process_adapter_has_a_stable_callable_boundary() {
        let adapter: fn(&ProcessInvocation) -> io::Result<ExitStatus> = status;
        let _ = adapter;
    }
}
