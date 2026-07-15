use std::ffi::OsString;
use std::io::Write;

use crate::cli::Cli;
use crate::commands;
use crate::output::{Output, OutputOptions};

const EXIT_SUCCESS: u8 = 0;
const EXIT_FAILURE: u8 = 1;
const EXIT_USAGE: u8 = 2;

/// Parses CLI arguments, dispatches the selected command and translates the
/// result into a process exit code.
pub fn run<I, S, Stdout, Stderr>(args: I, stdout: &mut Stdout, stderr: &mut Stderr) -> u8
where
    I: IntoIterator<Item = S>,
    S: Into<OsString>,
    Stdout: Write,
    Stderr: Write,
{
    run_with_options(args, stdout, stderr, OutputOptions::default())
}

/// Runs the CLI with explicit presentation options.
pub fn run_with_options<I, S, Stdout, Stderr>(
    args: I,
    stdout: &mut Stdout,
    stderr: &mut Stderr,
    output_options: OutputOptions,
) -> u8
where
    I: IntoIterator<Item = S>,
    S: Into<OsString>,
    Stdout: Write,
    Stderr: Write,
{
    let cli = match Cli::parse(args) {
        Ok(cli) => cli,
        Err(error) => {
            if error.use_stderr() {
                let _ = write!(stderr, "{error}");
                return EXIT_USAGE;
            }

            let _ = write!(stdout, "{error}");
            return EXIT_SUCCESS;
        }
    };

    let mut output = Output::new(stdout, stderr, output_options);
    match commands::execute(cli.command, &mut output) {
        Ok(outcome) => outcome.exit_code(),
        Err(error) => {
            let _ = output.error(error);
            EXIT_FAILURE
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn runs_version_command_end_to_end() {
        let mut stdout = Vec::new();
        let mut stderr = Vec::new();

        let code = run(["version"], &mut stdout, &mut stderr);
        let output = String::from_utf8(stdout).unwrap();

        assert_eq!(code, EXIT_SUCCESS);
        assert!(output.starts_with(&format!(
            "{} version v{} ",
            javaup_core::PRODUCT_NAME,
            env!("CARGO_PKG_VERSION")
        )));
        assert!(output.ends_with(&format!("({})\n", env!("JAVAUP_BUILD_DATE"))));
        assert!(stderr.is_empty());
    }

    #[test]
    fn recognizes_init_command() {
        let cli = Cli::parse(["init"]).unwrap();
        assert_eq!(cli.command, crate::cli::Command::Init);
    }

    #[test]
    fn reports_usage_errors_to_stderr() {
        let mut stdout = Vec::new();
        let mut stderr = Vec::new();

        let code = run(["unknown"], &mut stdout, &mut stderr);
        let error = String::from_utf8(stderr).unwrap();

        assert_eq!(code, EXIT_USAGE);
        assert!(stdout.is_empty());
        assert!(error.contains("unrecognized subcommand 'unknown'"));
        assert!(error.contains("Usage: jup <COMMAND>"));
        assert!(error.contains("--help"));
    }

    #[test]
    fn shows_help_when_no_command_is_provided() {
        let mut stdout = Vec::new();
        let mut stderr = Vec::new();

        let code = run(std::iter::empty::<&str>(), &mut stdout, &mut stderr);
        let help = String::from_utf8(stderr).unwrap();

        assert_eq!(code, EXIT_USAGE);
        assert!(stdout.is_empty());
        assert!(help.contains(javaup_core::PRODUCT_DESCRIPTION));
        assert!(help.contains("Usage: jup <COMMAND>"));
        assert!(help.contains("init"));
        assert!(help.contains("build"));
        assert!(help.contains("version"));
        assert!(help.contains("--help"));
    }

    #[test]
    fn writes_help_to_stdout_as_a_successful_response() {
        let mut stdout = Vec::new();
        let mut stderr = Vec::new();

        let code = run(["--help"], &mut stdout, &mut stderr);
        let help = String::from_utf8(stdout).unwrap();

        assert_eq!(code, EXIT_SUCCESS);
        assert!(help.contains("Usage: jup <COMMAND>"));
        assert!(help.contains("init"));
        assert!(help.contains("build"));
        assert!(help.contains("version"));
        assert!(help.contains("--help"));
        assert!(stderr.is_empty());
    }

    #[test]
    fn writes_version_flag_to_stdout_as_a_successful_response() {
        let mut stdout = Vec::new();
        let mut stderr = Vec::new();

        let code = run(["--version"], &mut stdout, &mut stderr);
        let version = String::from_utf8(stdout).unwrap();

        assert_eq!(code, EXIT_SUCCESS);
        assert_eq!(
            version,
            format!("{} {}\n", "jup", env!("JAVAUP_CLI_VERSION"))
        );
        assert!(stderr.is_empty());
    }
}
