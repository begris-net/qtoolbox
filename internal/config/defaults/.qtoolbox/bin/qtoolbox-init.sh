#!/usr/bin/env bash

if [ -z "$QTOOLBOX_DIR" ]; then
    BIN_DIR="$( dirname -- "$0")"
	export QTOOLBOX_DIR="$( readlink -m "$BIN_DIR/..")"
fi

COMPLETION_FILE="$QTOOLBOX_DIR/var/_completion"
if [[ -n "$ZSH_VERSION" ]]; then
	$QTOOLBOX_DIR/bin/qtoolbox completion zsh > $COMPLETION_FILE
elif [[ -n "$BASH_VERSION" ]]; then
	$QTOOLBOX_DIR/bin/qtoolbox completion bash > $COMPLETION_FILE
fi

export PATH=$QTOOLBOX_DIR/bin:$PATH
source $COMPLETION_FILE
alias qtb=qtoolbox tb=qtoolbox

CANDIDATES_DIR="$QTOOLBOX_DIR/candidates"
# init installed candidates
for candidate_name in $(find $CANDIDATES_DIR -mindepth 1 -maxdepth 1 -type d -exec basename {} \;); do
    echo "candidate: $candidate_name"
	candidate_dir="${CANDIDATES_DIR}/${candidate_name}/current"
	if [[ -h "$candidate_dir" || -d "${candidate_dir}" ]]; then
#		__sdkman_export_candidate_home "$candidate_name" "$candidate_dir"
#		__sdkman_prepend_candidate_to_path "$candidate_dir"
        echo $candidate_dir
	fi
done
unset candidate_name candidate_dir
export PATH

function __qtoolbox() {
    cmd=$(command -v qtoolbox)
    eval $cmd "$@"
    local final_rc=$?

    return $final_rc
}

