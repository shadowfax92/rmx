function rmx --description 'rmx: short-verb wrapper around wrapux'
    if test (count $argv) -eq 0
        wrapux ls
        return
    end

    set -l rest $argv[2..]

    switch $argv[1]
        case l ls list
            wrapux ls $rest
        case a attach
            wrapux attach $rest
        case c cap capture
            wrapux capture $rest
        case rm k kill remove
            wrapux rm $rest
        case '*'
            wrapux $argv
    end
end
