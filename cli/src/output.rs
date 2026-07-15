use std::fmt;
use std::io::{self, Write};

const ANSI_RESET: &str = "\u{1b}[0m";
const ANSI_CYAN: &str = "\u{1b}[36m";
const ANSI_GREEN: &str = "\u{1b}[32m";
const ANSI_YELLOW: &str = "\u{1b}[33m";
const ANSI_RED: &str = "\u{1b}[31m";

/// Controls presentation details for CLI status messages.
#[derive(Clone, Copy, Debug, Default, Eq, PartialEq)]
pub struct OutputOptions {
    status_color: bool,
}

impl OutputOptions {
    /// Emits colored status labels. The destination stream is responsible for
    /// adapting or stripping ANSI styles according to terminal capabilities.
    pub fn colored() -> Self {
        Self { status_color: true }
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

    pub(crate) fn info(&mut self, message: impl fmt::Display) -> io::Result<()> {
        write_status(
            self.stderr,
            self.options.status_color,
            "INFO",
            ANSI_CYAN,
            message,
        )
    }

    pub(crate) fn success(&mut self, message: impl fmt::Display) -> io::Result<()> {
        write_status(
            self.stderr,
            self.options.status_color,
            "SUCCESS",
            ANSI_GREEN,
            message,
        )
    }

    pub(crate) fn warning(&mut self, message: impl fmt::Display) -> io::Result<()> {
        write_status(
            self.stderr,
            self.options.status_color,
            "WARNING",
            ANSI_YELLOW,
            message,
        )
    }

    pub(crate) fn error(&mut self, message: impl fmt::Display) -> io::Result<()> {
        write_status(
            self.stderr,
            self.options.status_color,
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
        let mut output = Output::new(&mut stdout, &mut stderr, OutputOptions::colored());

        output.success("done").unwrap();
        output.warning("fallback").unwrap();
        output.error("failed").unwrap();

        assert_eq!(
            String::from_utf8(stderr).unwrap(),
            "\u{1b}[32m[SUCCESS]\u{1b}[0m done\n\u{1b}[33m[WARNING]\u{1b}[0m fallback\n\u{1b}[31m[ERROR]\u{1b}[0m failed\n"
        );
    }
}
