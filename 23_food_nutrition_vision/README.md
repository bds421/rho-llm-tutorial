# Tutorial 23: food-photo nutrition evaluation

A single-photo vision CLI using rho-llm v0.7.5. It forces one `report_nutrition` function call, validates the returned kcal/macronutrient fields, and optionally compares them with reference values. It is an evaluation building block for both datasets below, not a dataset parser, training pipeline, or validated dietary assessment system.

## Run

Requires Go 1.26.8 and an exported `GEMINI_API_KEY`. No `.env` is loaded automatically. Run this only after deciding the image may be sent to your selected provider. The command performs one paid inference request with retries disabled, a default 60-second deadline (maximum 2 minutes), and a 1,024-token output limit.

```sh
cd 23_food_nutrition_vision
go run . -image /absolute/path/to/meal.jpg
# Illustrative reference values only: replace all four with this photo's labels.
go run . -image /absolute/path/to/meal.jpg -calories 600 -fat 20 -carbs 75 -protein 30
# Alternate provider: supply a full model ID supporting vision and forced tools.
go run . -image /absolute/path/to/meal.png -provider anthropic -model claude-haiku-4-5 -api-key-env ANTHROPIC_API_KEY
```

Only local JPEG/PNG files up to 4 MiB and 25 megapixels are accepted; invalid/truncated images fail before upload. Resize larger originals locally and record that preprocessing. The response is a JSON object with `calories_kcal`, `fat_g`, `carbs_g`, `protein_g`, and `assumptions`, preceded by a caution and followed by optional errors. Missing, negative, nonfinite, extra, or incomplete tool results fail rather than silently becoming zeros. Ground-truth flags never enter the model prompt.

Absolute error is `abs(prediction - reference)`; percent error is `100 * absolute / reference`. Zero reference produces `N/A`, not infinity. This reports per-photo errors, not aggregate dataset metrics. For a benchmark record sample ID, image view, dataset version/split, provider/model/version, prompt, preprocessing, prediction, reference, failures, and zero-reference exclusions. Report sample counts and MAE per nutrient; distinguish mean per-item percentage error from error normalized by dataset mean, and follow the dataset's official evaluation protocol for comparable results.

## Nutrition5k

[Official dataset and download instructions](https://github.com/google-research-datasets/Nutrition5k) describe **5,006 plates** from a few **California cafeterias**, ingredient masses and dish-level calories/macros, RGB/depth data and videos, **official train/test splits**, and a full archive of approximately **181.4 GB**. The repository was archived read-only on April 19, 2026, but remains the canonical static documentation and links to the Google Cloud data. Data are **CC BY 4.0**. Cite Thames et al., *Nutrition5k: Towards Automatic Nutritional Understanding of Generic Food*, CVPR 2021 ([paper](https://arxiv.org/abs/2103.03375)).

Download is opt-in: open the official repository's Download Data section, inspect available storage, and choose individual RGB images plus metadata/split files from its linked Google Cloud bucket instead of automatically fetching the full archive. Keep media outside Git or in ignored `data/`; no download command runs as part of this tutorial. Match each selected image's dish ID to its dish-level kcal/fat/carbs/protein labels, then pass those values to the CLI. Use the official held-out split and keep every view/frame of one dish in the same split. A single RGB view does not reproduce the paper's multi-view/depth evaluation. Do not tune prompts on the test set.

## SNAPMe

[Official USDA Ag Data Commons dataset](https://agdatacommons.nal.usda.gov/articles/dataset/SNAPMe_A_Benchmark_Dataset_of_Food_Photos_with_Food_Records_for_Evaluation_of_Computer_Vision_Algorithms_in_the_Context_of_Dietary_Assessment/24856449): **3,311 photos**, **275 ASA24 records**, **95 participants**, approximately **1.89 GB**, **CC BY-SA 4.0**. Use it for **benchmark/evaluation only; it must not be used to train models** in this tutorial, consistent with the publisher's stated use limitation. This no-training guidance is separate from the Creative Commons license. Cite Chin et al. (2022), *SNAPMe: A Benchmark Dataset of Food Photos with Food Records for Evaluation of Computer Vision Algorithms in the Context of Dietary Assessment*, Ag Data Commons ([DOI](https://doi.org/10.15482/USDA.ADC/1528346)), and the [study paper](https://pmc.ncbi.nlm.nih.gov/articles/PMC10708545/). [Official analysis code](https://github.com/JulesLarke-USDA/SNAPMe) provides the original evaluation context.

Download is opt-in: visit the dataset record, review its license/use limitations, manually download the archive only if wanted, and read its included README and ASA24 linkage documentation before choosing photos. Never auto-download or commit its media. Use suitable before-meal photos for visible-serving estimates. Do not assign a whole-day ASA24 total to one photo, count before/after photos as independent meals, or treat packaging labels as a photographed serving. Establish the relevant food-item/meal linkage and portion basis before supplying reference flags; when consumed intake differs from the pictured serving, this CLI's single-image prediction is not directly comparable. Keep participants/days and paired images together in evaluation accounting. No training or prompt-tuning split is created from SNAPMe.

## Ethics and limitations

Photo estimates are **not medical advice** and must not guide insulin dosing, treatment, or clinical nutrition decisions. Visible appearance cannot reliably reveal portion mass, oils, recipe ingredients, or leftovers; model numbers may be confidently wrong. Dataset coverage, cafeteria/cultural bias, ASA24 self-report uncertainty, and possible pretraining contamination limit conclusions. These tests establish code behavior, not nutritional accuracy.

Respect participant privacy and dataset terms; avoid uploading identifying content, location metadata, or sensitive meal records without an appropriate basis. Use a provider/account configuration that does not use submitted SNAPMe evaluation data for training, or an appropriate local deployment; investigate provider retention before upload. Preserve attribution/license notices and applicable ShareAlike obligations when redistributing adaptations. Report failures and selection criteria rather than cherry-picking successful images. No images, labels, or benchmark scores ship with this module.

## Offline checks

```sh
go test -race -count=1 ./...
go vet ./...
```

Tests generate tiny synthetic images locally and use a fake client. They never call providers or download either dataset.
