# links

To make things easier with an idea that may be unhappy with extra copies
of key directories, I symlink these directories to the tinygo
sources.  Set TINYGOSRC to where your tinygo source is located.

* ln -s $TINYGOSRC/src/runtime tinygo_runtime
* ln -s $TINYGOSRC/src/device .
* ln -s $TINYGOSRC/src/machine .

You want to call your link to the "runtime" `tinygo_runtime`
so as not confuse your idea with the standard go implementation
package "runtime".