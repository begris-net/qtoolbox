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


if [ -z "$QTOOLBOX_DIR" ]; then
    BIN_DIR="$( dirname -- "$0")"
    export QTOOLBOX_DIR="$( readlink -m "$BIN_DIR/..")"
fi

function __qtoolbox_pathadd {
    PATH=:$PATH
    PATH=$1${PATH//:$1:/:}
}

function __qtoolbox_debug {
    if [[ "$__QTOOLBOX_DEBUG" == "debug" ]]; then
        echo "$@"
    fi
}

__QTOOLBOX_POSTPROCESSING_FLAG="##POSTPROCESSING##"

function qtoolbox() {
    local ret args final_rc
    __qtoolbox_debug "qtoolbox wrapper" >&2;
    ret=$(command qtoolbox "$@")
    local final_rc=$?
    case "$ret" in
      *$__QTOOLBOX_POSTPROCESSING_FLAG*)
        args=($(echo $ret | tr " " "\n"))
        __qtoolbox_postprocessing $args;;
      *)
        echo $ret
    esac
    return $final_rc
}

function __qtoolbox_init() {
    QTOOLBOX_BIN_DIR=$QTOOLBOX_DIR/bin
    export QTOOLBOX_HOOK_DIR=$QTOOLBOX_DIR/hooks
    COMPLETION_FILE="$QTOOLBOX_DIR/var/_completion"
    if [[ -n "$ZSH_VERSION" ]]; then
        $QTOOLBOX_BIN_DIR/qtoolbox completion zsh > $COMPLETION_FILE
    elif [[ -n "$BASH_VERSION" ]]; then
        $QTOOLBOX_BIN_DIR/qtoolbox completion bash > $COMPLETION_FILE
    fi

    if [[ $PATH =~ ${QTOOLBOX_BIN_DIR} ]]; then
        export PATH=${PATH//${QTOOLBOX_BIN_DIR}/${QTOOLBOX_BIN_DIR}}
    else
        __qtoolbox_pathadd "$QTOOLBOX_BIN_DIR"
    fi

    source $COMPLETION_FILE
    alias qtb=qtoolbox tb=qtoolbox
    export QTOOLBOX_CANDIDATES_DIR="$QTOOLBOX_DIR/candidates"

    __qtoolbox_initialize_candidates $QTOOLBOX_CANDIDATES_DIR
}
#typeset -fTt __qtoolbox_init

function __qtoolbox_initialize_candidates() {
    local qtoolbox_candidates_dir candidate_name candidate_dir
    qtoolbox_candidates_dir=$1

    for candidate_name in $(find $qtoolbox_candidates_dir -mindepth 1 -maxdepth 1 -type d -exec basename {} \;); do
        __qtoolbox_debug "candidate: $candidate_name"
    	__qtoolbox_update_candidate_path "$candidate_name" current
    	__qtoolbox_set_candidate_home $candidate_name "current"
    done
}

function __qtoolbox_update_candidate_path() {
    local candidate version candidate_dir close_path candidate_bin
    candidate="$1"
    version="$2"
    candidate_dir="${QTOOLBOX_CANDIDATES_DIR}/${candidate}/${version}"
    __qtoolbox_debug $candidate_dir
    if [[ -h "$candidate_dir" || -d "${candidate_dir}" ]]; then
        candidate_bin=$($QTOOLBOX_BIN_DIR/qtoolbox candidate export "$candidate")

        if [[ $PATH =~ ${QTOOLBOX_CANDIDATES_DIR}\/${candidate}\/([^/:]+)([^:]*) ]]; then
            local matched_version match_path

            if [[ "$ZSH_VERSION" == "$($SHELL -c 'echo $ZSH_VERSION')" ]]; then
                matched_version=${match[1]}
                matched_path=${match[2]}
            else
                matched_version=${BASH_REMATCH[1]}
                matched_path=${BASH_REMATCH[2]}
            fi
            export PATH=${PATH//${QTOOLBOX_CANDIDATES_DIR}\/${candidate}\/${matched_version}${matched_path}/${QTOOLBOX_CANDIDATES_DIR}\/${candidate}\/${version}${matched_path}}
        else
            if [[ -n "$candidate_bin" ]]; then
                candidate_dir="$candidate_dir/$candidate_bin"
            fi
            __qtoolbox_pathadd "$candidate_dir"
        fi
    fi
    export PATH
}
#typeset -fTt __qtoolbox_update_candidate_path

function __qtoolbox_set_candidate_home() {
	local candidate version upper_candidate
	candidate="$1"
	version="$2"
	upper_candidate=$(echo "$candidate" | tr '[:lower:]' '[:upper:]')
	__qtoolbox_debug "${upper_candidate}_HOME=${QTOOLBOX_CANDIDATES_DIR}/${candidate}/${version}"
	export "${upper_candidate}_HOME"="${QTOOLBOX_CANDIDATES_DIR}/${candidate}/${version}"
}

function __qtoolbox_process_hook() {
    local command="$1"
    shift 1;

    if [[ $QTOOLBOX_HOOK_DEBUG ]]; then
        echo "[DEBUG] hook command: $command"
    fi

    if [[ -f "$QTOOLBOX_HOOK_DIR/$command" ]]; then
        if [[ $QTOOLBOX_HOOK_DEBUG ]]; then
            echo "[DEBUG] hook availabe: $QTOOLBOX_HOOK_DIR/$command"
        fi
        "$QTOOLBOX_HOOK_DIR/$command" $@
    fi
}

function __qtoolbox_postprocessing() {
    local flag command candidate version
    flag=$1
    shift 1;
    command=$1
    shift 1;
    candidate=$1
    version=$2

    if [[ $flag =~ $__QTOOLBOX_POSTPROCESSING_FLAG ]]; then
        if [[ -n "$command" && -n "$candidate" && -n "$version" ]]; then
            case $command in
                install|default)
                    __qtoolbox_update_candidate_path $candidate "current"
                    __qtoolbox_set_candidate_home $candidate "current";;
                use)
                    __qtoolbox_update_candidate_path $candidate $version
                    __qtoolbox_set_candidate_home $candidate $version;;
            esac

            __qtoolbox_process_hook $command $candidate $version
        fi
    fi
}

__qtoolbox_init
