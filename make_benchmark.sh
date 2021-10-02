#!/bin/zsh

output_folder=${2:-qbf}

rm -fr $output_folder
mkdir -p $output_folder

go build encode.go

#for enc in {143,16}
for enc in {143,16}
do 
    #for x in {gttt-3x3,gttt-4x4}
    #for x in {gttt-3x3,gttt-4x4,gttt-5x5-iterative-deepening,hex-hein,milestones}
    for x in {gttt-3x3,hex-hein}
    do 
            mkdir -p $output_folder/$enc 
            mkdir -p $output_folder/$enc/$x 
            for instance in $x/*.pg;
            do
             	a=$(mktemp)
                ./encode $instance --enc=$enc > $a
                #echo grounding $instance $enc 
                #bule ground bule/pg$enc.bul $a -t=0 | grep -v '^c ' > $output_folder/$enc/$x/$(basename $instance .pg).qdimacs
                echo bule ground bule/pg$enc.bul $a -t=0 '>' $output_folder/$enc/$x/$(basename $instance .pg).qdimacs
            done 
    done
done
