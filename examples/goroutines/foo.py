import sys

def print_odds(limit=10):
    """
    Print odds numbers < limit
    """
    for i in range(limit):
        if i%2:
            sys.stderr.write("odds: {}\n".format(i))

def print_even(limit=10):
    """
    Print even numbers < limit
    """
    for i in range(limit):
        if i%2 == 0:
            sys.stderr.write("even: {}\n".format(i))
