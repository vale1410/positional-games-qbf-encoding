#!/usr/bin/env python3

import sys
import os

def readAndExtend(game,l,key):
    while l and l[0] and l[0][0][0] != "#":
        line = l.pop(0)
        game[key].extend(line)
    return (game,l)
def readAndAppend(game,l,key):
    if key not in game:
        game[key] = []
    while l and l[0] and l[0][0][0] != "#":
        line = l.pop(0)
        game[key].append(line)
    return (game,l)
def flattenField(game,key):
    if key in game:
        game[key] = [x for xs in game[key] for x in xs]
    return game

def parse_file(f):
    l = []
    for line in f:
        strings = line.split()
        l.append(strings)
    l = [line for line in l if line and line[0][0] != "%"]
    game = dict()
    i = 0
    while l:
        line = l.pop(0)
        assert line and line[0]
        case = line[0]
        (game,l) = readAndAppend(game,l,case)
    game = flattenField(game,"#blackinitials")
    game = flattenField(game,"#whiteinitials")
    game = flattenField(game,"#times")
    game = flattenField(game,"#blackturns")
    game = flattenField(game,"#positions")
    game = flattenField(game,"#source")
    game = flattenField(game,"#target")
    if "#blackwins" in game:
        hyper = [[str(i),v] for i,h in enumerate(game["#blackwins"]) for v in h]
        game["#blackwins"] = hyper
    return game

def uncapitalize(s):
    assert s
    return s[0].lower() + s[1:]
def uncapitalizes(l):
    return [uncapitalize(s) for s in l]

def make_unary_output(g,pg,bule):
    result = ""
    if pg in g and g[pg]:
        facts = ['{}[{}]'.format(bule,uncapitalize(f)) for f in g[pg]]
        result = "#ground {}.".format(', '.join(facts))
    return result
def make_list_output(g,pg,bule):
    result = ""
    if pg in g:
        facts = ['{}[{}]'.format(bule,','.join(uncapitalizes(f))) for f in g[pg]]
        result = "#ground {}.".format(', '.join(facts))
    return result

def output_game(game):
    blackinit = make_unary_output(game,"#blackinitials","blackinit")
    whiteinit = make_unary_output(game,"#whiteinitials","whiteinit")
    times = make_unary_output(game,"#times","times")
    times = '#ground final[{}].'.format(len(game['#times']))
    vertices = make_unary_output(game,"#positions","vertex")
    hyperedges = make_list_output(game,"#blackwins","hyperedge")
    source = make_unary_output(game,"#source","source")
    target = make_unary_output(game,"#target","target")
    edges = make_list_output(game,"#edges","edge")
    result = '\n'.join([blackinit,whiteinit,times,vertices,hyperedges,edges,source,target])
    return result

def process_file(testname):
    with open("./" + testname) as in_file:
        game = parse_file(in_file)
        out = output_game(game)
        return out

if __name__ == "__main__":
    if len(sys.argv) not in [2]:
        print("usage: `script.py file`")
    else:
        print(process_file(sys.argv[1]))
