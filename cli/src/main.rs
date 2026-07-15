use std::process::ExitCode;

fn main() -> ExitCode {
    let stdout = std::io::stdout();
    let stderr = anstream::AutoStream::new(std::io::stderr(), anstream::ColorChoice::Auto);
    let mut stdout = stdout.lock();
    let mut stderr = stderr.lock();

    let code = cli::run_with_options(
        std::env::args_os().skip(1),
        &mut stdout,
        &mut stderr,
        cli::OutputOptions::colored(),
    );
    ExitCode::from(code)
}
