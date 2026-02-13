ALTER TABLE profiles
ADD COLUMN IF NOT EXISTS zodiac TEXT;

UPDATE profiles
SET zodiac = CASE
    WHEN ((EXTRACT(MONTH FROM birthdate) = 3 AND EXTRACT(DAY FROM birthdate) >= 21)
       OR (EXTRACT(MONTH FROM birthdate) = 4 AND EXTRACT(DAY FROM birthdate) <= 19)) THEN 'aries'
    WHEN ((EXTRACT(MONTH FROM birthdate) = 4 AND EXTRACT(DAY FROM birthdate) >= 20)
       OR (EXTRACT(MONTH FROM birthdate) = 5 AND EXTRACT(DAY FROM birthdate) <= 20)) THEN 'taurus'
    WHEN ((EXTRACT(MONTH FROM birthdate) = 5 AND EXTRACT(DAY FROM birthdate) >= 21)
       OR (EXTRACT(MONTH FROM birthdate) = 6 AND EXTRACT(DAY FROM birthdate) <= 20)) THEN 'gemini'
    WHEN ((EXTRACT(MONTH FROM birthdate) = 6 AND EXTRACT(DAY FROM birthdate) >= 21)
       OR (EXTRACT(MONTH FROM birthdate) = 7 AND EXTRACT(DAY FROM birthdate) <= 22)) THEN 'cancer'
    WHEN ((EXTRACT(MONTH FROM birthdate) = 7 AND EXTRACT(DAY FROM birthdate) >= 23)
       OR (EXTRACT(MONTH FROM birthdate) = 8 AND EXTRACT(DAY FROM birthdate) <= 22)) THEN 'leo'
    WHEN ((EXTRACT(MONTH FROM birthdate) = 8 AND EXTRACT(DAY FROM birthdate) >= 23)
       OR (EXTRACT(MONTH FROM birthdate) = 9 AND EXTRACT(DAY FROM birthdate) <= 22)) THEN 'virgo'
    WHEN ((EXTRACT(MONTH FROM birthdate) = 9 AND EXTRACT(DAY FROM birthdate) >= 23)
       OR (EXTRACT(MONTH FROM birthdate) = 10 AND EXTRACT(DAY FROM birthdate) <= 22)) THEN 'libra'
    WHEN ((EXTRACT(MONTH FROM birthdate) = 10 AND EXTRACT(DAY FROM birthdate) >= 23)
       OR (EXTRACT(MONTH FROM birthdate) = 11 AND EXTRACT(DAY FROM birthdate) <= 21)) THEN 'scorpio'
    WHEN ((EXTRACT(MONTH FROM birthdate) = 11 AND EXTRACT(DAY FROM birthdate) >= 22)
       OR (EXTRACT(MONTH FROM birthdate) = 12 AND EXTRACT(DAY FROM birthdate) <= 21)) THEN 'sagittarius'
    WHEN ((EXTRACT(MONTH FROM birthdate) = 12 AND EXTRACT(DAY FROM birthdate) >= 22)
       OR (EXTRACT(MONTH FROM birthdate) = 1 AND EXTRACT(DAY FROM birthdate) <= 19)) THEN 'capricorn'
    WHEN ((EXTRACT(MONTH FROM birthdate) = 1 AND EXTRACT(DAY FROM birthdate) >= 20)
       OR (EXTRACT(MONTH FROM birthdate) = 2 AND EXTRACT(DAY FROM birthdate) <= 18)) THEN 'aquarius'
    WHEN ((EXTRACT(MONTH FROM birthdate) = 2 AND EXTRACT(DAY FROM birthdate) >= 19)
       OR (EXTRACT(MONTH FROM birthdate) = 3 AND EXTRACT(DAY FROM birthdate) <= 20)) THEN 'pisces'
    ELSE zodiac
END,
updated_at = NOW()
WHERE birthdate IS NOT NULL
  AND (zodiac IS NULL OR TRIM(zodiac) = '');
