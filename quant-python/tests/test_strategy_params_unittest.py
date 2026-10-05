import unittest

from app.strategies.engine import required_bars


class StrategyParameterTests(unittest.TestCase):
    def test_rejects_unknown_parameter_instead_of_silently_using_default(self):
        with self.assertRaisesRegex(ValueError, "fast_period"):
            required_bars("ma_cross", {"fast_period": 5})

    def test_rejects_fractional_integer_parameter(self):
        with self.assertRaisesRegex(ValueError, "period"):
            required_bars("rsi_reversal", {"period": 14.5})

    def test_accepts_supported_parameter_set(self):
        self.assertEqual(required_bars("ma_cross", {"short_window": 8, "long_window": 34}), 35)


if __name__ == "__main__":
    unittest.main()
