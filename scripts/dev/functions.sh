#!/bin/bash

println() {
    local color
    bold=$(tput bold)
    reset=$(tput sgr0)

    case "$1" in
    "red") color=$(tput setaf 1) ;;
    "green") color=$(tput setaf 2) ;;
    "yellow") color=$(tput setaf 3) ;;
    "blue") color=$(tput setaf 4) ;;
    "magenta") color=$(tput setaf 5) ;;
    "cyan") color=$(tput setaf 6) ;;
    "white") color=$(tput setaf 7) ;;
    *) color="\e[39m" ;;
    esac

    if [ -z "$3" ]; then
        echo -e "${color}${2}${reset}"
    else
        echo -e "${bold}${color}${2}${reset}"
    fi
}

exitStateBack() {
    if [ $? -ne 0 ]; then
        println "red" "执行失败 ==> $1 \n" >&2
        exit 1
    fi
}

exitState() {
    local exit_code=$?
    local message=${1:-"命令执行失败"}

    if [ $exit_code -ne 0 ]; then
        echo "错误: $message (退出码: $exit_code)" >&2
        exit $exit_code
    fi
}

exitPrintln() {
    println "red" "$1" >&2
    exit 1
}

# 转大写字母
capitalize() {
    echo "$1" | awk '{print toupper(substr($0, 1, 1)) substr($0, 2)}'
}

# 转小写字母
lowercase() {
    echo "$(echo "${1}" | tr '[:upper:]' '[:lower:]')"
}

# 首字母大写
toPascalCase() {
    local str="$1"
    echo "$(echo "${str:0:1}" | tr '[:lower:]' '[:upper:]')${str:1}"
}

# 获取指定参数
get_specific_parameter() {
    param_name=$1
    shift
    while [ $# -gt 0 ]; do
        if [ "$1" == "$param_name" ]; then
            shift
            echo $1
            return 0
        fi
        shift
    done

    return 1
}
