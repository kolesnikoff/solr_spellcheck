SOLR_SPELLCHECK
===============

Utility is used for spellchecking words into Solr's synonyms vocabulary.
It works in interaction with user and user may select proper value of the suggested word.
 
Logic of the script:

1. Replace digraphs (two letters) to one grapheme according to current languages rules (f.x. for Danish replace "oe" to "ø")
2. Check spelling.
3. If spelling is wrong - suggest proper spelling.
4. User should accept some version or skip word.

To run script you need to install aspell library and vocabularies.

    sudo apt-get install aspell-da aspell-hr aspell-hu aspell-nl aspell-no aspell-pl aspell-ro

This library has been used for the script: https://github.com/trustmaster/go-aspell 
It contains information how to install necessary modules for Go.

*Usage:**

    ./solr_spellcheck input_file locale
    
