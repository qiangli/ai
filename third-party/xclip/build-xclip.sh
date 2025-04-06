#!/bin/bash

# https://www.x.org/archive/individual/lib/libXmu-1.2.1.tar.xz
function build_libxmu() {
	wget https://www.x.org/archive/individual/lib/libXmu-1.2.1.tar.xz
	tar -xf libXmu-1.2.1.tar.xz
	cd libXmu-1.2.1 || exit 1
	./configure --prefix="${HOME}/local"
	make
	make install || { echo "Failed to install libXmu"; exit 1; }	
}

# https://github.com/astrand/xclip.git
function build_xclip() {
    git clone https://github.com/astrand/xclip.git
	cd xclip || exit 1
	./bootstrap
	./configure --prefix="${HOME}/local"
	make
	make install || { echo "Failed to install xclip"; exit 1; }
}

##
mkdir -p "${HOME}/local/build"

cd "${HOME}/local/build" || exit 1
build_libxmu

export CFLAGS="-I${HOME}/local/include"
export LDFLAGS="-L${HOME}/local/lib"
export PKG_CONFIG_PATH="${HOME}/local/lib/pkgconfig:${HOME}/local/share/pkgconfig"

cd "${HOME}/local/build" || exit 1
build_xclip || { echo "Failed to build xclip"; exit 1; }

echo "Done building xclip"
##