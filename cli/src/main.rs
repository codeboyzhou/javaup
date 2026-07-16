use std::process::ExitCode;

fn main() -> ExitCode {
    let stdout = anstream::AutoStream::new(std::io::stdout(), anstream::ColorChoice::Auto);
    let stderr = anstream::AutoStream::new(std::io::stderr(), anstream::ColorChoice::Auto);
    let mut stdout = stdout.lock();
    let mut stderr = stderr.lock();

    let code = javaup_cli::run_with_options(
        std::env::args_os().skip(1),
        &mut stdout,
        &mut stderr,
        javaup_cli::OutputOptions::styled(),
    );
    ExitCode::from(code)
}
