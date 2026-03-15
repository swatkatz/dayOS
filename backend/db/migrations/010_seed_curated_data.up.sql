-- Clear existing context entries and routines, then seed with curated data

DELETE FROM context_entries;
DELETE FROM routines;

-- Context entries
INSERT INTO context_entries (id, category, key, value, is_active) VALUES
  ('34bde9d5-c11c-49ee-92eb-5b149ed0566f', 'life', 'baby', '6-month-old daughter. Nanny present 9am-4pm Mon-Fri. Best quality time after naps (~10am, ~2pm). Also need to take care of the baby from 4pm to 4:30 pm once the nanny leaves', true),
  ('f1026394-9f2e-485c-a2c3-680dc94e12d8', 'preferences', 'babybaton', 'Baby baton is my passion project and I want to continue working on that as time permits. I should dedicate about 5-7 hrs on it per week unless my interview is this week or next week. Try and schedule time for baby baton whenever possible. Here is the github project: https://github.com/swatkatz/babybaton', true),
  ('2eaf9938-6e7a-4528-8402-834702c8bb75', 'constraints', 'dinner_prep', 'Dinner prep needs to happen either at around 16:30, if we don''t go for a walk or later around 18:00 if we do end up going for a walk', true),
  ('2da192f3-8ae7-4af9-988f-1b98169beabd', 'constraints', 'energy', 'New parent. Cap deep cognitive work at 5h/day max. Hardest tasks go first in the morning.', true),
  ('f08f5063-5724-412c-9cf9-5cb566a6ce6a', 'constraints', 'evening_window', 'Baby typically asleep by ~7:30pm. After cleanup, there is a light evening window (~20:00-22:00). This time is primarily for rest and unwinding — do NOT schedule deep focus work here. All routines during this time should be honored', true),
  ('38ccf217-f94a-4ec1-afbb-902c3d442125', 'life', 'family', 'Partner Elijah. Baby. We also have 3 step kids who don''t eat Indian food. They are with us Monday, and Tue nights and Fri, Sat, Sun every other week. Indian food is made usually on Wed, and Thu when the kids are not with us', true),
  ('b75a875e-3c7f-40b8-8a82-267759c3ea77', 'equipment', 'groceries', 'Can order from instacart for meals for the week if meal_prep for the week is done on Sunday night', true),
  ('0c0bdca2-080a-4447-b4d7-4d9e27135d59', 'equipment', 'kitchen', 'Full kitchen setup. All standard utensils. We have Indian setup too. Instapot, tawa, Airfryer, oven, dutch oven, cast iron, etc. Also have Indian spices and western spices like italian seasoning, oregano, chilli flakes, paprika. Also have rus-al-hanood for moroccon dishes.', true),
  ('0e181c36-b05b-445a-95e3-49432a1fb84e', 'constraints', 'location', 'Toronto, Canada.', true),
  ('b1399e0d-dd95-4b4f-b150-dce22d25a68a', 'constraints', 'mat_leave', 'Currently on mat_leave till Mar 31st. Will have to go back to full time work as a Sr. Staff SWE at Pinterest starting Apr 1. There will be way less flexibility in the schedule when that happens. Weekends are still free.', true),
  ('1922f7b7-d6d7-49fc-8593-d7cd485f4902', 'preferences', 'planning_style', 'Time-blocked days. Buffers between intense sessions. Interview prep is non-negotiable daily.', true),
  ('4586359e-e6bd-4f53-92c4-9f6f171a1012', 'constraints', 'work_window', 'Deep focus work: 9am-4pm only (nanny present). Before 9 am some baby time, after 4:00 pm, baby time till 4:30 pm.', true);

-- Routines
INSERT INTO routines (id, title, category, frequency, days_of_week, preferred_time_of_day, preferred_duration_min, notes, is_active, preferred_exact_time) VALUES
  ('f49e60a9-6914-4d64-8aef-a7361561e5ec', 'Breast milk pump afternoon', 'BABY', 'DAILY', NULL, NULL, 20, 'afternoon pump', true, '13:00'),
  ('c072bcd5-d530-40dd-afb1-5d5f3e8e87b0', 'Breast milk pump evening', 'BABY', 'DAILY', NULL, NULL, 20, 'Evening pump', true, '17:00'),
  ('56c6b679-579b-4bed-b88b-8c0755824035', 'Breast milk pump morning', 'BABY', 'DAILY', NULL, NULL, 30, 'morning pumps are the best in terms of volume', true, '08:30'),
  ('375c54ba-11fd-42e3-ad04-a0a3b88b97d4', 'Breast milk pump night', 'BABY', 'DAILY', NULL, NULL, 30, 'night pump before sleep', true, '22:00'),
  ('2dc9e909-f3a8-4973-bb29-059945d77b4b', 'F45 at Ossington (Sunday)', 'EXERCISE', 'CUSTOM', '{0}', NULL, 75, '15 mins for travel, 45 mins exercise, and another 15 to travel back', true, '09:30'),
  ('e9304301-cad5-4267-aadf-4c3fe1e13dab', 'F45 at Ossington (weekdays)', 'EXERCISE', 'CUSTOM', '{3,5}', NULL, 75, '15 mins for travel, 45 mins exercise, and another 15 to travel back.', true, '09:00'),
  ('c2c75563-e40e-47e2-b58f-c3bcba03aa3b', 'Get ready', 'ADMIN', 'DAILY', NULL, NULL, 45, 'get ready for the day, make coffee, etc', true, '08:00'),
  ('78ab3f02-a7f6-440f-a886-8752f7d6d1cc', 'Lunch', 'MEAL', 'WEEKDAYS', NULL, 'MIDDAY', 30, 'Eat lunch', true, NULL),
  ('c77ffd35-c23d-4345-bfe0-695664c0e400', 'Soccer Monday (with Al)', 'EXERCISE', 'CUSTOM', '{1}', NULL, 90, 'Travel (20 mins), play (1 hr), travel back (20 mins) -- with Al, or uber', true, '18:40');
