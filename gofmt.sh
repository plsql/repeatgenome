# A shell two-liner to defy gc's attempt to force me to use tabs by removing the -tabs option from gofmt.

gofmt -s=true -w=true *.go repeatgenome/*.go
perl -pi -e 's/\t/    /g' *.go repeatgenome/*.go
