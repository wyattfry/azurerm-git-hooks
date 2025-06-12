#!/bin/bash

FILE="$1"
EXITCODE=0

# Functions to check
FUNCTIONS='HasChange|HasChanges|GetOk|GetChange|Set|Get'

# Extract all field names passed as the first string argument to the functions above
FIELDS=$(grep -oE "($FUNCTIONS)\(\"[^\"]+\"" "$FILE" | sed -E 's/.*\("([^"]+)".*/\1/')

# Also, for HasChanges, which can take multiple arguments, extract all string arguments
FIELDS_HASCHANGES=$(grep -oE 'HasChanges\([^)]+\)' "$FILE" | \
  grep -oE '"[^"]+"' | sed 's/"//g')

# Combine all fields and remove duplicates
FIELDS=$(echo -e "$FIELDS\n$FIELDS_HASCHANGES" | sort | uniq | grep -v '^$')

if [ -z "$FIELDS" ]; then
    exit 0
fi

for FIELD in $FIELDS; do
    if ! grep -Eq "tfschema:\"$FIELD\"" "$FILE" && ! grep -Eq "\"$FIELD\"\s*:\s*{" "$FILE"; then
        echo "ERROR: Field '$FIELD' referenced in SDK call but not found as tfschema or map key in $FILE"
        EXITCODE=1
    fi
done

exit $EXITCODE