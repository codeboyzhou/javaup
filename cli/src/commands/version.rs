use std::io::{self, Write};

use crate::build_info::BuildInfo;

pub(super) fn execute<W>(stdout: &mut W) -> io::Result<()>
where
    W: Write,
{
    writeln!(
        stdout,
        "{} version {}",
        javaup_core::PRODUCT_NAME,
        BuildInfo::current()
    )
}
