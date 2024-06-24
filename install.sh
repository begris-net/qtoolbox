#!/usr/bin/env bash
#
# Copyright (c) 2024 Bjoern Beier.
#
# Permission is hereby granted, free of charge, to any person obtaining a copy of
# this software and associated documentation files (the "Software"), to deal in
# the Software without restriction, including without limitation the rights to
# use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
# the Software, and to permit persons to whom the Software is furnished to do so,
# subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in all
# copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
# FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
# COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
# IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
# CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
#
UNAME=$(command -v uname)
ARCHIVE=''
QTB_BINARY='qtoolbox'
case $("${UNAME}" | tr '[:upper:]' '[:lower:]') in
    linux*)
        OS='linux'
        ARCHIVE='tar.gz'
        ;;
    darwin*)
        OS='darwin'
        ARCHIVE='tar.gz'
        ;;
    msys*|cygwin*|mingw*|nt|win*)
        OS='windows'
        ARCHIVE='zip'
        QTB_BINARY='qtoolbox.exe'
        ;;
    *)
        OS='unknown'
        ;;
esac

case $($UNAME -m | tr '[:upper:]' '[:lower:]') in
    x86_64)
        ARCH='amd64'
        ;;
    aarch64)
        ARCH='arm64'
        ;;
esac

QTOOLBOX_ARCHIVE_FILE=$(mktemp --suffix=$ARCHIVE)
LATEST_VERSION=$(curl -Ls -o /dev/null -w %{url_effective} https://github.com/begris-net/qtoolbox/releases/latest | grep -oE "[^/]+$")
DOWNLOAD_URL="https://github.com/begris-net/qtoolbox/releases/download/$LATEST_VERSION/qtoolbox-$LATEST_VERSION-$OS-$ARCH.$ARCHIVE"
curl -L $DOWNLOAD_URL -o $QTOOLBOX_ARCHIVE_FILE

EXTRACT_DIR=$(dirname $QTOOLBOX_ARCHIVE_FILE)
case "$ARCHIVE" in
    tar*)
        tar -C $EXTRACT_DIR -xf $QTOOLBOX_ARCHIVE_FILE $QTB_BINARY
        ;;
    zip)
        unzip -juod $EXTRACT_DIR $QTOOLBOX_ARCHIVE_FILE $QTB_BINARY
        ;;
esac

$EXTRACT_DIR/$QTB_BINARY setup "$@"
