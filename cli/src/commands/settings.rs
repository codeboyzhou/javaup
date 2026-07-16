use std::env;
use std::io::{self, Write};

use javaup_core::maven_settings::MavenSettingsProfile;
use javaup_core::project::ProjectEnvironment;

use crate::cli::SettingsCommand;
use crate::output::Output;

pub(super) fn execute<Stdout, Stderr>(
    command: SettingsCommand,
    output: &mut Output<'_, Stdout, Stderr>,
) -> io::Result<()>
where
    Stdout: Write,
    Stderr: Write,
{
    match command {
        SettingsCommand::Add { name, path } => {
            let profile = MavenSettingsProfile::register(name, path).map_err(io::Error::other)?;
            output.success(format_args!(
                "Registered Maven settings '{}': {}",
                profile.name(),
                profile.path().display()
            ))
        }
        SettingsCommand::List => {
            let profiles = MavenSettingsProfile::list().map_err(io::Error::other)?;
            if profiles.is_empty() {
                writeln!(output.stdout(), "No Maven settings profiles registered.")?;
            } else {
                for profile in profiles {
                    output.field(profile.name(), profile.path().display())?;
                }
            }
            output.stdout().flush()
        }
        SettingsCommand::Use { name } => {
            let profile = MavenSettingsProfile::resolve(name).map_err(io::Error::other)?;
            let current_dir = env::current_dir()?;
            let (project_dir, mut environment) =
                ProjectEnvironment::load_nearest(current_dir).map_err(io::Error::other)?;
            environment.set_maven_settings(Some(&profile));
            environment.save(&project_dir).map_err(io::Error::other)?;
            output.success(format_args!(
                "Bound Maven settings '{}' to {}",
                profile.name(),
                project_dir.display()
            ))
        }
        SettingsCommand::Clear => {
            let current_dir = env::current_dir()?;
            let (project_dir, mut environment) =
                ProjectEnvironment::load_nearest(current_dir).map_err(io::Error::other)?;
            let Some(profile_name) = environment.maven().settings_profile().map(str::to_owned)
            else {
                return output.warning(format_args!(
                    "No Maven settings profile is bound to {}",
                    project_dir.display()
                ));
            };
            environment.set_maven_settings(None);
            environment.save(&project_dir).map_err(io::Error::other)?;
            output.success(format_args!(
                "Cleared Maven settings '{profile_name}' from {}",
                project_dir.display()
            ))
        }
        SettingsCommand::Remove { name } => {
            let profile = MavenSettingsProfile::remove(name).map_err(io::Error::other)?;
            output.success(format_args!(
                "Removed Maven settings profile '{}'",
                profile.name()
            ))?;
            output.warning(format_args!(
                "Existing project bindings to '{}' were not changed",
                profile.name()
            ))
        }
    }
}
