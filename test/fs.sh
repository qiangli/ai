#!/usr/bin/env ai /sh:bash --format raw --script

echo "fs:* toolkit tests"

set -ue

##
/fs:help
/fs:list_roots

/fs:create_directory --option path="/tmp/test/dir"

/fs:write_file --option path="/tmp/test/dir/x" \
    --option content="this\nis\na\ncool\nfile\ntest\n"
/fs:copy_file --option source="/tmp/test/dir/x" --option destination="/tmp/test/dir/y"
/fs:move_file --option source="/tmp/test/dir/y" --option destination="/tmp/test/dir/z"
/fs:edit_file --option destination="/tmp/test/dir/z" \
    --option find="a" --option replace="one"  --option all_occurrences=true --option regex=false

/fs:list_directory --option path="/tmp/test/dir"
/fs:tree --option path="/tmp/test"

/fs:get_file_info --option path="/tmp/test/dir/z"
/fs:read_files --option path="/tmp/test/dir/z" \
    --option number=2 --option offset=2 --option limit=3
/fs:read_multiple_files --option paths="[/tmp/test/dir/x,/tmp/test/dir/z]"

/fs:search_files --option path="/tmp/test/" --option pattern="^fi*" 

/fs:delete_file --option path="/tmp/test/dir" \
    --option recursive=true
#
/sh:exec --command "ls -al /tmp/test/dir"

echo "*** File System tests completed ***"
###