from test_dtmf import string_to_dtmf, spell_string
df= [["John","Doe"], ["Kim", "Strohm"],["sarah","Nolan"]]

bot_name = "Bradley Scott"

def getname(x,text):
    s="Can you connect me to "
    y="Hello! This is"+" "+ bot_name+".Can you connect me to"+" "+(df[0][0]+" "+df[0][1])

    if x=="agent_conv":
        s=s+ df[0][0]+" "+df[0][1]
        return s

    elif x=="dialby_firstonly":
        if "spell" in text:
            return spell_string(df[0][0])+" "+df[0][0]
        else:
            return (df[0][0])
            
    elif x=="dialby_lastonly":
        if "spell" in text:
            return spell_string(df[0][1])+" "+df[0][1]
        else:
            return (df[0][1])

    elif x=="dialby_firstlast":
        if "spell" in text:
            return spell_string(df[0][0]+df[0][1])+" "+df[0][0]+" "+df[0][1]
        else:
            return ((df[0][0]+" "+df[0][1]))

    elif x== "dialby_lastfirst":
        if "spell" in text:
            return spell_string(df[0][1]+" "+df[0][0])+" "+df[0][1]+" "+df[0][0]
        else:
            return ((df[0][1]+" "+df[0][0]))

    elif x == "whos_calling":
        return bot_name

    elif x == "from-start":
        return y
        
    else:
        pass

def getname1(x):

    if x=="dialby_firstonly":
        return string_to_dtmf(df[0][0])
            
    elif x=="dialby_lastonly":
        return string_to_dtmf(df[0][1])

    elif x=="dialby_firstlast":
        return (string_to_dtmf(df[0][0])+"w"+string_to_dtmf(df[0][1]))

    elif x== "dialby_lastfirst":
        return (string_to_dtmf(df[0][1])+"w"+string_to_dtmf(df[0][0]))

        
    else:
        pass