# NOTE:
# Starting from PR939, the logic of the `release` target in the Makefile has changed.
# This script is used as a temporary workaround to perform local release operations
# until the main Makefile is fully revised.

git show 13282c6:Makefile > Makefile.release
sed -i '' 's/\$(MAKE)/$(MAKE) -f Makefile.release/g' Makefile.release
make -f Makefile.release release
rm Makefile.release
