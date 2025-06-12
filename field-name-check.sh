#!/bin/bash
set -eou pipefail

# Some functions in the Terraform SDK that take as arguments the names of configuration fields will silently fail if
# given input that does not match the name of any fields in the resource. This script errors out if the string
# arguments to these functions do not match any field names in the same file. Intended to be used on "*_resource.go" files.
#
#
# Caveats:
# 1. Only performs crude checks on nested properties, i.e. ensures that every nested property is found somewhere in the
#    resource definition. It does NOT check proper nesting relationship.
# 2. Only works if the function is provided strings. It will not work if the code is using a format string or variable.

FILE="$1"
EXITCODE=0

# All relevant functions
FUNCTIONS='HasChange|HasChanges|GetOk|GetChange|Set|Get'

# Extract all field strings given as first argument to those functions
FIELDS=$(grep -oE "($FUNCTIONS)\(\"[^\"]+\"" "$FILE" | sed -E 's/.*\("([^"]+)".*/\1/')

# Handle HasChanges with multiple arguments
FIELDS_HASCHANGES=$(grep -oE 'HasChanges\([^)]+\)' "$FILE" | grep -oE '"[^"]+"' | sed 's/"//g' || true)

# Combine and dedup fields
FIELDS=$(echo -e "$FIELDS\n$FIELDS_HASCHANGES" | sort | uniq | grep -v '^$')

if [ -z "$FIELDS" ]; then
    exit 0
fi

for FIELD in $FIELDS; do
    # Split on dots, skip numeric components
    COMPONENTS=$(echo $FIELD | tr '.' '\n' | grep -vE '^[0-9]+$')
    for COMP in $COMPONENTS; do
        # Errors out if the string argument to HasChanges() etc is NOT found in any of the following three cases:
        # 1. tfschema:"STRING"      typed resource with single-field annotation
        # 2. tfschema:"STRING,      typed resource with multi-field annotation
        # 3. "STRING": {            untyped (older) resource
        if ! grep -Eq "tfschema:\"${COMP}[\"\,]" "$FILE" && ! grep -Eq "\"$COMP\"\s*:\s*{" "$FILE"; then
            echo "ERROR: Component '$COMP' from field '$FIELD' is not found as tfschema or map key in $FILE"
            EXITCODE=1
        fi
    done
done

exit $EXITCODE