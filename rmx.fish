function rmx --description 'rmx: short-verb shortcuts'
    if test (count $argv) -eq 0
        command rmx ls
        return
    end

    set -l rest $argv[2..]

    switch $argv[1]
        case l ls list
            command rmx ls $rest
        case a attach
            command rmx attach $rest
        case c cat cap capture
            command rmx cat $rest
        case t tail follow
            command rmx tail $rest
        case s send
            command rmx send $rest
        case text enter
            command rmx send $argv
        case e exit quit
            command rmx exit $rest
        case clr clear
            command rmx clear $rest
        case rm k kill remove
            command rmx rm $rest
        case '*'
            command rmx $argv
    end
end
