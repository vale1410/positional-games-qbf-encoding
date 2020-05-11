# Instances and Encoder for Positional Games

Positional games are a mathematical class of two-player games comprising
Tic-tac-toe and its generalizations. We propose a novel encoding of these games
into \acp{QBF} such that a game instance admits a winning strategy for first player 
if and only if the corresponding formula is true. Our approach improves over
previous \ac{QBF} encodings of games in multiple ways. 
First, it is generic and lets us encode other positional games, such as Hex. 
Second, structural properties of positional games together with a careful
treatment of illegal moves let us generate more compact instances that can be
solved faster by state-of-the-art \ac{QBF} solvers. We establish the latter
fact through extensive experiments.
Finally, the compactness of our new encoding makes it feasible to translate realistic
game problems.
We identify a few such problems of historical significance and put them forward
to the \ac{QBF} community as milestones of increasing difficulty. 

## Generation of Benchmark


The benchmarks can be generated and placed in the folder `qbf` with the command:

```
./make_benchmark.sh
```

## Positional Game Description 1.0


The general encoder reads a game description $\prod = \langle T_\black,
T_\white,\depth,\size,E_\black, E_\white \rangle $ and produces a QDIMACS file that can
be passed to a \ac{QBF} solver. We briefly explain the format in which the game is
specified. Files have the file type \texttt{.pg}. 

A line in the file  is either a codeword that starts with $\#$, 
a list of vertices or time points separated by white space. A vertex and time
points must be alphanumeric strings. After each code word the program expects one or more
lines, each consisting of a list of vertices or time points separated by white space. 
The lines are read until the next code word or EOF (end of file). 

Code word | Game Specification | Comment
----------|---------------------|--------
#times} | $T$ & List of time points in the order of the game.  
#blackturns} | $T_\black$ & List of time points in which black plays. (Whites time points $T_\white = T\setminus T_\black$) 
#positions} | $V$ & The vertex set given as a list of vertices.  
#blackwins} | $E_\black$ & Each line consists of a list of vertices that define one winning configuration for black 
#whitewins} | $E_\white$ & Analog to \texttt{\#blackwins} 
#blackinitials} | -  & List of vertices that black owns before the game starts 
#whiteinitials} | -  & Analog to \texttt{\#blackinitials} 
#firstmoves} | - |  List of vertices that can be chosen from in the first move. \
#version} | | Version number of the game description. Currently 1.0. 
```

### Examples from the paper in Section 2. 

``` 
#version
1.0
#times
t4 t5 t6 t7 t8 t9
#blackturns
t5 t7 t9
#positions
a1 a2 a3 
b1 b2 b3
c1 c2 c3
#blackwins
a1 b1 c1
a2 b2 c2
a3 b3 c3
a1 a2 a3
b1 b2 b3
c1 c2 c3
a1 b2 c3
a3 b2 c1
#blackinitials
b2 c3
#whitewins
a1 b1 c1
a2 b2 c2
a3 b3 c3
a1 a2 a3
b1 b2 b3
c1 c2 c3
a1 b2 c3
a3 b2 c1
#whiteinitials
a1
```



