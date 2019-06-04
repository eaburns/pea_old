#!/bin/sh
o=$(mktemp tmp.XXXXXXXXXX)

fail() {
	echo Failed
	cat $o | grep -v deprecated
	rm $o
	exit 1
}

trap fail INT TERM

echo gofmt
gofmt -s -l $(find . -name '*.go') > $o 2>&1
test $(wc -l $o | awk '{ print $1 }') = "0" || fail

echo govet
go vet ./... > $o 2>&1 || fail

echo ineffassign
ineffassign . \
	| grep -v "grammar.go:" \
	> $o 2>&1
e=$(mktemp tmp.XXXXXXXXXX)
touch $e
diff $o $e > /dev/null || { rm $e; fail; }
rm $e

echo misspell
misspell . > $o 2>&1 || fail

echo gocyclo
gocyclo -over 15 .\
	| grep -v "grammar.go:" \
	| grep -v "findFun pea/check.go"\
	| grep -v "TestUnify pea/check_test.go"\
	> $o 2>&1
e=$(mktemp tmp.XXXXXXXXXX)
touch $e
diff $o $e > /dev/null || { rm $e; fail; }
rm $e

echo go test
go test -test.timeout=60s ./... > $o 2>&1 || fail

echo golint
golint ./... \
	| grep -v "grammar.go:" \
	| egrep -v "ast.go:.*(End|ExprType|Mod|Name|Priv|Start) should have comment" \
	> $o 2>&1
# Silly: diff the grepped golint output with empty.
# If it's non-empty, error, otherwise succeed.
e=$(mktemp tmp.XXXXXXXXXX)
touch $e
diff $o $e > /dev/null || { rm $e; fail; }
rm $e

rm $o
