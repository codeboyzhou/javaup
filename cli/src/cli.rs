use std::ffi::OsString;
use std::iter;

use clap::{Args, Parser, Subcommand};

const CLI_NAME: &str = "jup";

/// Command-line interface for managing Java installations.
#[derive(Debug, Parser)]
#[command(
    name = CLI_NAME,
    version = env!("JAVAUP_CLI_VERSION"),
    about = javaup_core::PRODUCT_DESCRIPTION,
    arg_required_else_help = true
)]
pub(crate) struct Cli {
    #[command(subcommand)]
    pub(crate) command: Command,
}

impl Cli {
    pub(crate) fn parse<I, S>(args: I) -> Result<Self, clap::Error>
    where
        I: IntoIterator<Item = S>,
        S: Into<OsString>,
    {
        Self::try_parse_from(
            iter::once(OsString::from(CLI_NAME)).chain(args.into_iter().map(Into::into)),
        )
    }
}

#[derive(Clone, Debug, Eq, PartialEq, Subcommand)]
pub(crate) enum Command {
    /// Detect the current Maven project and record its required environment.
    Init,
    /// Run Maven with the JDK recorded by `jup init`.
    ///
    /// Uses Maven Wrapper when configured by `jup init`.
    #[command(disable_help_flag = true)]
    Mvn(MvnArgs),
    /// Show the environment recorded for the current project.
    Status,
    /// Print version, platform and build information.
    Version,
}

#[derive(Clone, Debug, Eq, PartialEq, Args)]
pub(crate) struct MvnArgs {
    /// Maven goals and options passed through unchanged.
    #[arg(trailing_var_arg = true, allow_hyphen_values = true)]
    pub(crate) maven_arguments: Vec<OsString>,
}

#[cfg(test)]
mod tests {
    use clap::error::ErrorKind;

    use super::*;

    #[test]
    fn parses_init_subcommand() {
        let cli = Cli::parse(["init"]).unwrap();
        assert_eq!(cli.command, Command::Init);
    }

    #[test]
    fn parses_version_subcommand() {
        let cli = Cli::parse(["version"]).unwrap();
        assert_eq!(cli.command, Command::Version);
    }

    #[test]
    fn parses_status_subcommand() {
        let cli = Cli::parse(["status"]).unwrap();
        assert_eq!(cli.command, Command::Status);
    }

    #[test]
    fn parses_mvn_arguments() {
        let cli = Cli::parse(["mvn", "test", "-DskipTests"]).unwrap();
        assert_eq!(
            cli.command,
            Command::Mvn(MvnArgs {
                maven_arguments: vec!["test".into(), "-DskipTests".into()],
            })
        );
    }

    #[test]
    fn accepts_mvn_without_adding_goals() {
        let cli = Cli::parse(["mvn"]).unwrap();
        assert_eq!(
            cli.command,
            Command::Mvn(MvnArgs {
                maven_arguments: Vec::new(),
            })
        );
    }

    #[test]
    fn forwards_maven_help_flags() {
        for argument in ["-h", "--help"] {
            let cli = Cli::parse(["mvn", argument]).unwrap();
            assert_eq!(
                cli.command,
                Command::Mvn(MvnArgs {
                    maven_arguments: vec![argument.into()],
                })
            );
        }
    }

    #[test]
    fn generates_help_for_help_flags_and_missing_commands() {
        for argument in ["-h", "--help"] {
            let error = Cli::parse([argument]).unwrap_err();
            assert_eq!(error.kind(), ErrorKind::DisplayHelp);
            assert!(error.to_string().contains("Usage: jup <COMMAND>"));
        }

        let error = Cli::parse(std::iter::empty::<&str>()).unwrap_err();
        assert_eq!(
            error.kind(),
            ErrorKind::DisplayHelpOnMissingArgumentOrSubcommand
        );
        assert!(error.to_string().contains("Usage: jup <COMMAND>"));
    }

    #[test]
    fn describes_mvn_as_a_wrapper_aware_maven_command() {
        let error = Cli::parse(["help", "mvn"]).unwrap_err();
        assert_eq!(error.kind(), ErrorKind::DisplayHelp);
        let help = error.to_string();
        assert!(help.contains("Run Maven with the JDK recorded by `jup init`"));
        assert!(help.contains("Uses Maven Wrapper when configured by `jup init`"));
        assert!(help.contains("Maven goals and options passed through unchanged"));
    }

    #[test]
    fn generates_version_for_version_flags() {
        for argument in ["-V", "--version"] {
            let error = Cli::parse([argument]).unwrap_err();
            assert_eq!(error.kind(), ErrorKind::DisplayVersion);
            assert_eq!(
                error.to_string(),
                format!("{} {}\n", CLI_NAME, env!("JAVAUP_CLI_VERSION"))
            );
        }
    }

    #[test]
    fn rejects_unknown_commands_and_extra_arguments() {
        let error = Cli::parse(["install"]).unwrap_err();
        assert_eq!(error.kind(), ErrorKind::InvalidSubcommand);

        let error = Cli::parse(["build"]).unwrap_err();
        assert_eq!(error.kind(), ErrorKind::InvalidSubcommand);

        let error = Cli::parse(["version", "extra"]).unwrap_err();
        assert_eq!(error.kind(), ErrorKind::UnknownArgument);
        assert!(error.to_string().contains("Usage: jup version"));
    }
}
