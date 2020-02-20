#!/bin/sh

root="$PWD/lib"
peac="$root/peac"

go build -o $peac ./peac

find $root -name [0-9]*.go | xargs rm

for d in `go run pealist/pealist.go $root`; do
	if test -d $root/$d; then
		/bin/echo -n "$d "
		cd $root/$d
		out="$root/$d/out.test"
		$peac -o $out -test -root $root -path $d . || {
			echo fail to build
			cd $root/..
			exit 1
		}
		tmp=$(mktemp tmp.XXXXXXXXXX)
		if $out >& $tmp; then
			empty=$(mktemp tmp.XXXXXXXXXX)
			touch $empty
			if diff $tmp $empty 2>&1 > /dev/null; then
				echo ?
			else
				echo ok
			fi
			rm $empty
		else
			echo failed
			cat $tmp | sed 's/^Test /	/g'
		fi
		rm $tmp
	fi
done;

cd $root/..