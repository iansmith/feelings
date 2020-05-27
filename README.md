# feelings

## path and environment variables 
When working on feelings, it is useful to have an "enable script."

An enable script is one that you source and that sets
all the necessary evironment varibales.  You don't have to do
this, but a sample one is checked in as `enable-feelings.sample`.

You must _source_ this file into your shell, not try to run it.

## Makefiles and README.md 
Whenever you find a Makefile or a README.md you can be sure it applies
to *only* the directory that contains it.  Further, Makefiles expect
that you run `make` your current working directory equal to the 
directory that contains the Makefile.
 
