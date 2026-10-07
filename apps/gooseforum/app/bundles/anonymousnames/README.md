# Anonymous name components

`phrase6-v1` embeds the repository-authored [component lists](data/phrases.json) in
one binary. Every new name freely combines a two-character action, a two-character
scene/object, `的`, and a one-character animal. All components are Chinese Han characters.
The 36 actions, 163 scenes/objects and 32 animals produce 187,776 distinct six-character
names, including 躲进云里的猫, 抱着松果的熊 and 躲进松果的猫. No grammatical or semantic
compatibility filter limits the playful combinations.

Cryptographic random indices sample the complete product uniformly; a batch contains
10 distinct names. Names need not be unique across batches or personas. Changing the
component version never rewrites a confirmed name or a persisted candidate batch.
Candidates from an older source remain selectable until their existing expiry.
