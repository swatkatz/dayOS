-- Revert curated seed data (deletes seeded rows, does not restore previous data)
DELETE FROM routines WHERE id IN (
  'f49e60a9-6914-4d64-8aef-a7361561e5ec',
  'c072bcd5-d530-40dd-afb1-5d5f3e8e87b0',
  '56c6b679-579b-4bed-b88b-8c0755824035',
  '375c54ba-11fd-42e3-ad04-a0a3b88b97d4',
  '2dc9e909-f3a8-4973-bb29-059945d77b4b',
  'e9304301-cad5-4267-aadf-4c3fe1e13dab',
  'c2c75563-e40e-47e2-b58f-c3bcba03aa3b',
  '78ab3f02-a7f6-440f-a886-8752f7d6d1cc',
  'c77ffd35-c23d-4345-bfe0-695664c0e400'
);

DELETE FROM context_entries WHERE id IN (
  '34bde9d5-c11c-49ee-92eb-5b149ed0566f',
  'f1026394-9f2e-485c-a2c3-680dc94e12d8',
  '2eaf9938-6e7a-4528-8402-834702c8bb75',
  '2da192f3-8ae7-4af9-988f-1b98169beabd',
  'f08f5063-5724-412c-9cf9-5cb566a6ce6a',
  '38ccf217-f94a-4ec1-afbb-902c3d442125',
  'b75a875e-3c7f-40b8-8a82-267759c3ea77',
  '0c0bdca2-080a-4447-b4d7-4d9e27135d59',
  '0e181c36-b05b-445a-95e3-49432a1fb84e',
  'b1399e0d-dd95-4b4f-b150-dce22d25a68a',
  '1922f7b7-d6d7-49fc-8593-d7cd485f4902',
  '4586359e-e6bd-4f53-92c4-9f6f171a1012'
);
