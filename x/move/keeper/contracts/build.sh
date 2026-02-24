#!/usr/bin/env bash

if ! command -v initiad &> /dev/null
then 
    echo "initiad could not be found"
    exit
fi

initiad move build  --named-addresses 'TestAccount=0x2'
rm -rf ../binaries
mkdir ../binaries

find build/test1/bytecode_modules -type f -name "*.mv"  -depth 1 -exec cp {} ../binaries \;
find build/test1/bytecode_scripts -type f -name "*.mv"  -depth 1 -exec cp {} ../binaries \;