use std::env;
use std::io::{self, Write};

use javaup_core::project::ProjectEnvironment;

use crate::output::Output;

pub(super) fn execute<Stdout, Stderr>(output: &mut Output<'_, Stdout, Stderr>) -> io::Result<()>
where
    Stdout: Write,
    Stderr: Write,
{
    let current_dir = env::current_dir()?;
    let (project_dir, environment) =
        ProjectEnvironment::load_nearest(&current_dir).map_err(io::Error::other)?;
    let configuration_path =
        ProjectEnvironment::configuration_path(&project_dir).map_err(io::Error::other)?;
    let maven_source = if environment.maven().uses_wrapper() {
        "wrapper"
    } else {
        "PATH"
    };
    let java_version = environment
        .installed_java_version()
        .map_err(io::Error::other)?;

    output.field("Project", project_dir.display())?;
    output.field("Config", configuration_path.display())?;
    output.field("Build system", environment.project_type().as_str())?;
    output.field(
        "Maven version",
        format_args!("{} ({maven_source})", environment.maven().version()),
    )?;
    output.field("Java version", java_version)?;
    output.field("Java home", environment.java_home().display())
}
