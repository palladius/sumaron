# SUMARON 🐴👁️

A folder-based summarization tool powered by LLMs, designed to be fast and cache-efficient. 

The name **Sumaron** is a humorous pun: "Sumar" means donkey (somaro) in the Ferrarese/Emilian dialect, and "-on" is the augmentative suffix (making it "Big Donkey"), which phonetically sounds like Sauron, the Dark Lord of Mordor.

---

## Logo / Mascot Candidates 🎨

Here are 3 options for the project's mascot/logo. Click the links below to view the images:

### Option 1: The Dark Lord Donkey
* **Mascot Concept**: Sauron reimagined as a majestic donkey wearing spiked dark metal armor standing in Mordor, with a donkey-shaped Eye of Sauron in the sky.
* **Image File**: [sumaron_dark_lord.jpg](assets/sumaron_dark_lord.jpg)
* **Prompt**:
  > A funny high-fantasy digital art piece of Sauron from Lord of the Rings, but reimagined as a donkey. The donkey is wearing spiked dark metal armor like Sauron, standing dramatically on a volcanic ridge under a red-orange sky of Mordor. In the background, the glowing fiery Eye of Sauron is shaped like a giant donkey's head with long ears, casting a comedic red light on the volcanic ash landscape.

### Option 2: The Eye of Sumaron
* **Mascot Concept**: Barad-dûr with a giant, goofy, wide-eyed donkey eye looking around from the top.
* **Image File**: [eye_of_sumaron.jpg](assets/eye_of_sumaron.jpg)
* **Prompt**:
  > A comedic high-fantasy illustration of the dark tower Barad-dur, but instead of the terrifying Eye of Sauron, the giant glowing eye at the top of the tower is a hilarious, wide-eyed cartoonish donkey eye looking around. The scene is dramatic with dark clouds, volcanic smoke, and glowing red lava, but the funny donkey eye makes it absurd and lighthearted.

### Option 3: Sumaron's Feast
* **Mascot Concept**: Sumaron the donkey sitting on a dark lord throne eating traditional Ferrarese food (*cappellacci di zucca* and *Coppia Ferrarese* bread).
* **Image File**: [sumaron_throne.jpg](assets/sumaron_throne.jpg)
* **Prompt**:
  > A funny digital painting of a donkey sitting on a dark, imposing dark lord throne inside a volcanic castle hall. The donkey is wearing a spiky dark metal crown/helmet. On its lap is a plate of cappellacci di zucca (pumpkin pasta) and it is happily chewing on a piece of traditional Coppia Ferrarese bread. Fiery lava rivers glow in the background.

---

## Project Goal
I want to create a summarizer called SUMARON which summarizes via LLM a folder. Since this takes time, it will leave a `.sumaron.json` which contains a timestamp of summarization and some sort of md5 of the summarized content. The timestamp + hash should allow the system to know if we need to re-summarize it.

It uses `GEMINI_API_KEY` from the environment; if not found, it complains and asks for either the env var or the `--key KEY` option (with `--help` support). It will track the summarized folders in a determined place, e.g., `~/.sumaron-cache.json` or `~/.sumaron/` if it's more complex.

Since LLMs are slow, it is absolutely fundamental that we do NOT repeat the same operation twice, so if it is called a second time, it should say "cache hit" and replicate the result. Finally, we will support only a select number of formats, e.g., `.md`, `.html`, and `.json` to start with.

