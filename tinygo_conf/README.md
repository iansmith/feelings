# tinygo-conf

Configuration files that are used by any build stage are kept in this directory.
This is keep them centralized and make it clear _which_ file is being used by
any given build step.

Any additional assembly files used by any build stage are assemblied into
this directory's child, `assembly`.  This is, again, for centralization but
also to make it easy to be _sure_ that you have cleaned up all the old
assembly files when you want to do a clean build.
