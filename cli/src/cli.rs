use std::ffi::OsString;
use std::iter;

use clap::{Parser, Subcommand};

/// Command-line interface for managing Java installations.
#[derive(Debug, Parser)]
#[command(
    name = javaup_core::PRODUCT_NAME,
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
            iter::once(OsString::from(javaup_core::PRODUCT_NAME))
                .chain(args.into_iter().map(Into::into)),
        )
    }
}

#[derive(Clone, Copy, Debug, Eq, PartialEq, Subcommand)]
pub(crate) enum Command {
    /// Detect the current Maven project and record its required environment.
    Init,
    /// Print version, platform and build information.
    Version,
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
    fn generates_help_for_help_flags_and_missing_commands() {
        for argument in ["-h", "--help"] {
            let error = Cli::parse([argument]).unwrap_err();
            assert_eq!(error.kind(), ErrorKind::DisplayHelp);
            assert!(error.to_string().contains("Usage: javaup <COMMAND>"));
        }

        let error = Cli::parse(std::iter::empty::<&str>()).unwrap_err();
        assert_eq!(
            error.kind(),
            ErrorKind::DisplayHelpOnMissingArgumentOrSubcommand
        );
        assert!(error.to_string().contains("Usage: javaup <COMMAND>"));
    }

    #[test]
    fn generates_version_for_version_flags() {
        for argument in ["-V", "--version"] {
            let error = Cli::parse([argument]).unwrap_err();
            assert_eq!(error.kind(), ErrorKind::DisplayVersion);
            assert_eq!(
                error.to_string(),
                format!(
                    "{} {}\n",
                    javaup_core::PRODUCT_NAME,
                    env!("JAVAUP_CLI_VERSION")
                )
            );
        }
    }

    #[test]
    fn rejects_unknown_commands_and_extra_arguments() {
        let error = Cli::parse(["install"]).unwrap_err();
        assert_eq!(error.kind(), ErrorKind::InvalidSubcommand);

        let error = Cli::parse(["version", "extra"]).unwrap_err();
        assert_eq!(error.kind(), ErrorKind::UnknownArgument);
        assert!(error.to_string().contains("Usage: javaup version"));
    }
}
