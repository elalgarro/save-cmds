if [ "$#" -eq 0 ]
then
    go run main.go 
    exit 0
fi

arg="$1"

case "$arg" in
    add | clear)
        go run main.go "$@" ;;
    * )
        eval $( go run main.go "$@" ) ;;
esac


