# antc

This is the board-side of the anticipation bootloader.

It is customary when copying-and-making-small-changes to the linker
scripts (`tinygo/targets/rpi3.ld` and `tinygo/targets/rpi3_qemu.ld`) to
name the changed linker scripts after _why_ you made the change not
the name of the target.  This is because linker scripts cannot "inherit"
or "override" other linker scripts and it is very easy at a glance to think
two linker scripts are the same.  Thus, if this directory had "rpi3.ld"
and "rpi3_qemu.ld" and you didn't notice that these two files differ
by one line from their brothers in `tinygo/targets`, you might decide
to delete them or otherwise botch things up.  The easiest way to express
why is just the name of the program that needs the changed linker script, 
as we have done here.

Similarly, but less tricky, is to do the same for the targets files,
such as `antc_qemu.json` and `antc.json.`  These files *can* override their
counterparts in the `tinygo/targets` directory, so the values in the files
should be only the ones that were needed.