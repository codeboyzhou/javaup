use std::fmt;
use std::io::{self, Write};

const ANSI_RESET: &str = "\u{1b}[0m";
const ANSI_BOLD: &str = "\u{1b}[1m";
const ANSI_CYAN: &str = "\u{1b}[36m";
const ANSI_GREEN: &str = "\u{1b}[32m";
const ANSI_YELLOW: &str = "\u{1b}[33m";
const ANSI_RED: &str = "\u{1b}[31m";

/// Controls presentation details for CLI output.
#[derive(Clone, Copy, Debug, Default, Eq, PartialEq)]
pub struct OutputOptions {
    styles_enabled: bool,
}

impl OutputOptions {
    /// Emits styled field and status labels. The destination stream is
    /// responsible for adapting or stripping ANSI styles according to terminal
    /// capabilities.
    pub fn styled() -> Self {
        Self {
            styles_enabled: true,
        }
    }
}

/// Owns the CLI's output policy so commands do not need to know about ANSI
/// styles, output destinations or flushing behavior.
pub(crate) struct Output<'a, Stdout, Stderr> {
    stdout: &'a mut Stdout,
    stderr: &'a mut Stderr,
    options: OutputOptions,
}

impl<'a, Stdout, Stderr> Output<'a, Stdout, Stderr>
where
    Stdout: Write,
    Stderr: Write,
{
    pub(crate) fn new(
        stdout: &'a mut Stdout,
        stderr: &'a mut Stderr,
        options: OutputOptions,
    ) -> Self {
        Self {
            stdout,
            stderr,
            options,
        }
    }

    pub(crate) fn stdout(&mut self) -> &mut Stdout {
        self.stdout
    }

    pub(crate) fn field(&mut self, label: &str, value: impl fmt::Display) -> io::Result<()> {
        if self.options.styles_enabled {
            write!(self.stdout, "{ANSI_BOLD}{label}:{ANSI_RESET}")?;
        } else {
            write!(self.stdout, "{label}:")?;
        }
        writeln!(self.stdout, " {value}")
    }

    pub(crate) fn info(&mut self, message: impl fmt::Display) -> io::Result<()> {
        write_status(
            self.stderr,
            self.options.styles_enabled,
            "INFO",
            ANSI_CYAN,
            message,
        )
    }

    pub(crate) fn success(&mut self, message: impl fmt::Display) -> io::Result<()> {
        write_status(
            self.stderr,
            self.options.styles_enabled,
            "SUCCESS",
            ANSI_GREEN,
            message,
        )
    }

    pub(crate) fn warning(&mut self, message: impl fmt::Display) -> io::Result<()> {
        write_status(
            self.stderr,
            self.options.styles_enabled,
            "WARNING",
            ANSI_YELLOW,
            message,
        )
    }

    pub(crate) fn error(&mut self, message: impl fmt::Display) -> io::Result<()> {
        write_status(
            self.stderr,
            self.options.styles_enabled,
            "ERROR",
            ANSI_RED,
            message,
        )
    }
}

fn write_status(
    writer: &mut impl Write,
    color_enabled: bool,
    label: &str,
    color: &str,
    message: impl fmt::Display,
) -> io::Result<()> {
    if color_enabled {
        write!(writer, "{color}[{label}]{ANSI_RESET}")?;
    } else {
        write!(writer, "[{label}]")?;
    }
    writeln!(writer, " {message}")?;
    writer.flush()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn writes_readable_statuses_without_color() {
        let mut stdout = Vec::new();
        let mut stderr = Vec::new();
        let mut output = Output::new(&mut stdout, &mut stderr, OutputOptions::default());

        output.info("detecting project").unwrap();
        output.success("project detected").unwrap();
        output.warning("wrapper not found").unwrap();
        output.error("detection failed").unwrap();

        assert_eq!(
            String::from_utf8(stderr).unwrap(),
            "[INFO] detecting project\n[SUCCESS] project detected\n[WARNING] wrapper not found\n[ERROR] detection failed\n"
        );
    }

    #[test]
    fn colors_only_status_labels() {
        let mut stdout = Vec::new();
        let mut stderr = Vec::new();
        let mut output = Output::new(&mut stdout, &mut stderr, OutputOptions::styled());

        output.success("done").unwrap();
        output.warning("fallback").unwrap();
        output.error("failed").unwrap();

        assert_eq!(
            String::from_utf8(stderr).unwrap(),
            "\u{1b}[32m[SUCCESS]\u{1b}[0m done\n\u{1b}[33m[WARNING]\u{1b}[0m fallback\n\u{1b}[31m[ERROR]\u{1b}[0m failed\n"
        );
    }

    #[test]
    fn emphasizes_field_labels_when_styled() {
        let mut stdout = Vec::new();
        let mut stderr = Vec::new();
        let mut output = Output::new(&mut stdout, &mut stderr, OutputOptions::styled());

        output.field("Java version", "17.0.12").unwrap();

        assert_eq!(
            String::from_utf8(stdout).unwrap(),
            "\u{1b}[1mJava version:\u{1b}[0m 17.0.12\n"
        );
        assert!(stderr.is_empty());
    }

    #[test]
    fn writes_plain_field_labels_without_styles() {
        let mut stdout = Vec::new();
        let mut stderr = Vec::new();
        let mut output = Output::new(&mut stdout, &mut stderr, OutputOptions::default());

        output.field("Java version", "17.0.12").unwrap();

        assert_eq!(
            String::from_utf8(stdout).unwrap(),
            "Java version: 17.0.12\n"
        );
        assert!(stderr.is_empty());
    }
}
