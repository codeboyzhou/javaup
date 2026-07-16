//! Process descriptions and execution ports shared by product use cases.

use std::ffi::{OsStr, OsString};
use std::io;
use std::path::{Path, PathBuf};
use std::process::{Command, ExitStatus};

/// A platform-native process invocation that frontends can execute using
/// their own streaming, cancellation, or asynchronous process facilities.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct ProcessInvocation {
    program: PathBuf,
    arguments: Vec<OsString>,
    current_dir: PathBuf,
    environment: Vec<(OsString, OsString)>,
}

impl ProcessInvocation {
    pub(crate) fn new(
        program: impl Into<PathBuf>,
        arguments: impl IntoIterator<Item = OsString>,
        current_dir: impl Into<PathBuf>,
    ) -> Self {
        Self {
            program: program.into(),
            arguments: arguments.into_iter().collect(),
            current_dir: current_dir.into(),
            environment: Vec::new(),
        }
    }

    pub(crate) fn set_env(&mut self, key: impl Into<OsString>, value: impl Into<OsString>) {
        let key = key.into();
        if let Some((_, existing)) = self
            .environment
            .iter_mut()
            .find(|(existing, _)| existing == &key)
        {
            *existing = value.into();
        } else {
            self.environment.push((key, value.into()));
        }
    }

    /// Changes only the process working directory, retaining its program,
    /// arguments, and selected Java environment.
    pub fn with_current_dir(mut self, current_dir: impl Into<PathBuf>) -> Self {
        self.current_dir = current_dir.into();
        self
    }

    pub fn program(&self) -> &Path {
        &self.program
    }

    pub fn arguments(&self) -> impl Iterator<Item = &OsStr> {
        self.arguments.iter().map(OsString::as_os_str)
    }

    pub fn current_dir(&self) -> &Path {
        &self.current_dir
    }

    pub fn environment(&self) -> impl Iterator<Item = (&OsStr, &OsStr)> {
        self.environment
            .iter()
            .map(|(key, value)| (key.as_os_str(), value.as_os_str()))
    }
}

/// Captured output produced by a process runner.
#[derive(Debug)]
pub struct ProcessOutput {
    pub(crate) status: ExitStatus,
    pub(crate) stdout: Vec<u8>,
    pub(crate) stderr: Vec<u8>,
}

impl ProcessOutput {
    /// Creates captured process output for a custom runner implementation.
    pub fn new(status: ExitStatus, stdout: Vec<u8>, stderr: Vec<u8>) -> Self {
        Self {
            status,
            stdout,
            stderr,
        }
    }

    pub fn status(&self) -> ExitStatus {
        self.status
    }

    pub fn stdout(&self) -> &[u8] {
        &self.stdout
    }

    pub fn stderr(&self) -> &[u8] {
        &self.stderr
    }
}

/// Port used by core detection and validation when output must be captured.
pub trait ProcessRunner {
    fn output(&self, invocation: &ProcessInvocation) -> io::Result<ProcessOutput>;
}

/// Blocking operating-system process adapter used by convenience APIs.
#[derive(Clone, Copy, Debug, Default)]
pub struct SystemProcessRunner;

impl ProcessRunner for SystemProcessRunner {
    fn output(&self, invocation: &ProcessInvocation) -> io::Result<ProcessOutput> {
        let output = command(invocation).output()?;
        Ok(ProcessOutput::new(
            output.status,
            output.stdout,
            output.stderr,
        ))
    }
}

fn command(invocation: &ProcessInvocation) -> Command {
    let mut command = Command::new(invocation.program());
    command
        .args(invocation.arguments())
        .current_dir(invocation.current_dir())
        .envs(invocation.environment());
    command
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn invocation_preserves_platform_native_process_data() {
        let mut invocation = ProcessInvocation::new(
            PathBuf::from("mvn"),
            [OsString::from("verify")],
            PathBuf::from("project"),
        );
        invocation.set_env("JAVA_HOME", "jdk-21");
        let invocation = invocation.with_current_dir("project/module");

        assert_eq!(invocation.program(), Path::new("mvn"));
        assert_eq!(
            invocation.arguments().collect::<Vec<_>>(),
            [OsStr::new("verify")]
        );
        assert_eq!(invocation.current_dir(), Path::new("project/module"));
        assert_eq!(
            invocation.environment().collect::<Vec<_>>(),
            [(OsStr::new("JAVA_HOME"), OsStr::new("jdk-21"))]
        );
    }
}
