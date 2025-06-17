import json

# Vehicles
with open('make-year-model.json') as f:
    data = json.load(f)

makes = set()
years = set()
models = set()
engines = set()
cars = set()

for make, ydict in data.items():
    makes.add(make)
    for year, mdict in ydict.items():
        years.add(year)
        for model, elist in mdict.items():
            models.add(model)
            for engine in elist:
                engines.add(engine)
                cars.add((make, year, model, engine))

# Parts
with open('part.json') as f:
    parts = json.load(f)

with open('seed_db.sql', 'w') as f:
    # Vehicles
    for m in sorted(makes):
        f.write(f"INSERT OR IGNORE INTO Make (name) VALUES ('{m.replace("'", "''")}');\n")
    for y in sorted(years, key=lambda x: int(x)):
        f.write(f"INSERT OR IGNORE INTO Year (year) VALUES ({int(y)});\n")
    for m in sorted(models):
        f.write(f"INSERT OR IGNORE INTO Model (name) VALUES ('{m.replace("'", "''")}');\n")
    for e in sorted(engines):
        f.write(f"INSERT OR IGNORE INTO Engine (name) VALUES ('{e.replace("'", "''")}');\n")
    for make, year, model, engine in sorted(cars):
        f.write(f"INSERT OR IGNORE INTO Car (make_id, year_id, model_id, engine_id) VALUES ((SELECT id FROM Make WHERE name='{make.replace("'", "''")}'), (SELECT id FROM Year WHERE year={int(year)}), (SELECT id FROM Model WHERE name='{model.replace("'", "''")}'), (SELECT id FROM Engine WHERE name='{engine.replace("'", "''")}'));\n")
    # Parts
    for cat in parts.keys():
        f.write(f"INSERT OR IGNORE INTO PartCategory (name) VALUES ('{cat.replace("'", "''")}');\n")
    for cat, subs in parts.items():
        for sub in subs:
            f.write(f"INSERT OR IGNORE INTO PartSubCategory (category_id, name) VALUES ((SELECT id FROM PartCategory WHERE name='{cat.replace("'", "''")}'), '{sub.replace("'", "''")}');\n") 