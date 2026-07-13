use std::env;
use std::io::{self, Write};

use javaup_core::project::ProjectEnvironment;

pub(super) fn execute<W>(stdout: &mut W) -> io::Result<()>
where
    W: Write,
{
    let project_dir = env::current_dir()?;
    let environment = ProjectEnvironment::detect(&project_dir).map_err(io::Error::other)?;
    let config_path = environment.save(&project_dir).map_err(io::Error::other)?;

    writeln!(
        stdout,
        "created {} (JDK {} at {}, Maven {}, wrapper: {})",
        config_path.display(),
        environment.java_version(),
        environment.java_home().display(),
        environment.maven().version(),
        environment.maven().uses_wrapper()
    )
}
