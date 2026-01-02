#!/usr/bin/env ai /sh:bash --format raw --script

echo "fs:* toolkit tests"

set -xue

##
# /fs:help
# /fs:list_roots

# /fs:create_directory --option path="/tmp/test/dir"
# content="
# This
# is
# the
# cool file
# test
# "
# /fs:write_file --option path="/tmp/test/dir/x" \
#     --option content="$content"
# /fs:copy_file --option source="/tmp/test/dir/x" --option destination="/tmp/test/dir/y"
# /fs:move_file --option source="/tmp/test/dir/y" --option destination="/tmp/test/dir/z"
# /fs:edit_file --option destination="/tmp/test/dir/z" \
#     --option find="^cool.*$" --option replace="coolest"  --option all_occurrences=true --option regex=true

# /fs:list_directory --option path="/tmp/test/dir"
# /fs:tree --option path="/tmp/test/dir" --option follow_symlinks=true --option depth=3

# /fs:get_file_info --option path="/tmp/test/dir/z"
# /fs:read_file --option path="/tmp/test/dir/z" \
#     --option number=2 --option offset=2 --option limit=3
# /fs:read_multiple_files --option paths='["/tmp/test/dir/x","/tmp/test/dir/z"]'

# /fs:search_files --option path="/tmp/test/dir" --option pattern="^fi*" 

# /fs:delete_file --option path="/tmp/test/dir" \
#     --option recursive=true
#
# /sh:exec --command "ls -al /tmp/test/"

echo "*** fs* toolkit tests completed ***"
###