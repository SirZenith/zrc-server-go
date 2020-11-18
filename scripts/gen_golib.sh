#!/bin/bash
######################################################
# build static lib for iOS using golib
######################################################
set -e

TOOL_NAME=$1
PROJ_ROOT=$(pwd)
MAKE_PATH=$PROJ_ROOT/theos_code
LIB_PATH=$PROJ_ROOT/.golib
LIB_NAME=libgolang.a

mkdir -p $LIB_PATH

export CGO_ENABLED=1
export GOARCH=arm64
export CC=$PROJ_ROOT/scripts/clangwrap.sh
export CXX=$PROJ_ROOT/scripts/clangwrap.sh

echo "building darwin/arm64 static lib"
go build -buildmode=c-archive -o $LIB_PATH/$LIB_NAME
if [ ! -f $LIB_PATH/$LIB_NAME ]; then
    echo "failed to build darwin/arm64 static lib!"
    exit 1
fi

######################################################
# build debian binary for iOS using theos
######################################################
# Makefile of .deb package
cd $MAKE_PATH
echo 'include $(THEOS)/makefiles/common.mk

export ARCHS = arm64

TOOL_NAME = '$TOOL_NAME'
'$TOOL_NAME'_FILES = main.mm
'$TOOL_NAME'_LDFLAGS = '$LIB_PATH/$LIB_NAME'

include $(THEOS_MAKE_PATH)/tool.mk
' > ./Makefile

rm -rf .theos
