import re

def string_to_dtmf(string):
    def letter_to_dtmf(letter):
        dtmf_tones = {
            '1': '1',
            '2': '2',
            '3': '3',
            'A': '2',
            '4': '4',
            '5': '5',
            '6': '6',
            'B': '2',
            '7': '7',
            '8': '8',
            '9': '9',
            'C': '2',
            '*': '*',
            '0': '0',
            '#': '#',
            'D': '3',
            'E': '3',
            'F': '3',
            'G': '4',
            'H': '4',
            'I': '4',
            'J': '5',
            'K': '5',
            'L': '5',
            'M': '6',
            'N': '6',
            'O': '6',
            'P': '7',
            'Q': '7',
            'R': '7',
            'S': '7',
            'T': '8',
            'U': '8',
            'V': '8',
            'W': '9',
            'X': '9',
            'Y': '9',
            'Z': '9'
        }
        return dtmf_tones.get(letter.upper(), '')
    
    return ''.join(letter_to_dtmf(letter) for letter in string)

def extract_number(string):
    
    numbers = re.findall(r'\d+', string)
    return (int(number) for number in numbers)

def spell_string(s):
  spelled_string = ""
  for char in s:
    spelled_string += char + "-"
  return spelled_string[:-1]

    

