#!/usr/bin/env python3

from multiprocessing import Pool
import subprocess
import functools
import time
import os
import statistics as st
import graph_to_bul as gtb

verbose = True

def converted_name(prefix, name):
    return "tmp/" + prefix + "/buleinput/" + name + ".bul"
def ground_name(prefix, name):
    return "output/" + prefix + "/" + name + ".dimacs"

def convert(prefix, instance_dir, name):
    input_file = instance_dir + "/" + name + ".pg"
    start_time = time.time()
    result = gtb.process_file(input_file)
    duration = time.time() - start_time
    filename = converted_name(prefix, name)
    os.makedirs(os.path.dirname(filename), exist_ok=True)
    with open(filename, "w") as outf:
        outf.write(result)
    if verbose:
        print("{} converted in {:.2f}s".format(name, duration))

def ground(prefix, encoding, name):
    test_case = converted_name(prefix, name)
    args = ["bule", "--output", "qdimacs", test_case] + encoding
    start_time = time.time()
    result = subprocess.run(args,text=True,capture_output=True)
    duration = time.time() - start_time
    filename = ground_name(prefix, name)
    os.makedirs(os.path.dirname(filename), exist_ok=True)
    with open(filename, "w") as outf:
        outf.write(result.stdout)
    if verbose:
        print("{} ground in {:.2f}s".format(name, duration))

def ground_all(prefix, instance_dir, encoding):
    print("for " + instance_dir + ", " + str(encoding) + " :")
    names = get_files_in_folder(instance_dir)
    start_time = time.time()
    with Pool(5) as p:
        p.map(functools.partial(convert, prefix, instance_dir), names)
    with Pool(5) as p:
        p.map(functools.partial(ground, prefix, encoding), names)
    duration = time.time() - start_time
    print("Finished grounding after {:.2f}s".format(duration))
    print()


def get_files_in_folder(folder_path):
    file_list = []
    for filename in os.listdir(folder_path):
        if os.path.isfile(os.path.join(folder_path, filename)):
            file_list.append(os.path.splitext(filename)[0])
    return file_list


if __name__ == "__main__":

    encoding_COR= ["bule/cor.bul"] 
    encoding_EA = ["bule/std_index.bul", "bule/explicit.bul", "bule/pg.bul"] 
    encoding_EN = ["bule/std_index.bul", "bule/explicit.bul", "bule/level.bul"]
    encoding_ET = ["bule/std_index.bul", "bule/explicit.bul", "bule/transversal.bul"]

    #for HEIN 
    ground_all("hein_EA", "./input/Hein/PG-format/", encoding_EA)
    ground_all("hein_ET", "./input/Hein/White_flipped/R-Gex_EGF-format/", encoding_ET)
    ground_all("hein_EN", "./input/Hein/R-Gex/EGF-format/", encoding_EN)
    #ground_all("hein_COR", "./input/Hein/PG-format/", encoding_COR)

    #for Championship SAT instances 
    ground_all("champion_SAT_ET", 
            "./input/Hex-Championship-2023-little-golem/unsat/R-Gex/EGF-format/",
            encoding_ET)
    ground_all("champion_SAT_EN", 
            "./input/Hex-Championship-2023-little-golem/unsat/White_flipped/R-Gex_EGF-format/",
            encoding_EN)

    #for Championship UNSAT instances
    ground_all("champion_UNSAT_ET", 
            "./input/Hex-Championship-2023-little-golem/unsat/R-Gex/EGF-format/",
            encoding_ET)
    ground_all("champion_UNSAT_EN", 
            "./input/Hex-Championship-2023-little-golem/unsat/White_flipped/R-Gex_EGF-format/",
            encoding_EN)

