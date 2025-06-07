import csv
import logging
from pathlib import Path
from typing import List, Dict, Any, Optional

logger = logging.getLogger(__name__)


class CSVExporter:
    """Handles exporting data to CSV files."""

    @staticmethod
    def export_to_csv(data: List[Dict[str, Any]], output_file: str) -> str:
        """Export a list of dictionaries to a CSV file.

        Args:
            data: List of dictionaries containing data to export
            output_file: Path to the output CSV file

        Returns:
            str: Path to the generated CSV file
        """
        if not data:
            logger.warning("No data to export to CSV")
            return ""

        # Ensure the output directory exists
        output_path = Path(output_file)
        output_path.parent.mkdir(parents=True, exist_ok=True)

        try:
            with open(output_path, 'w', newline='') as csvfile:
                if not data:
                    return str(output_path.absolute())

                fieldnames = list(data[0].keys())
                writer = csv.DictWriter(csvfile, fieldnames=fieldnames)
                writer.writeheader()
                writer.writerows(data)

            logger.info("Data exported to %s", output_path.absolute())
            return str(output_path.absolute())

        except Exception as e:
            logger.error("Error exporting to CSV: %s", str(e), exc_info=True)
            raise

    @classmethod
    def export_instances_to_csv(cls, instances: List[Dict[str, Any]],
                                output_file: Optional[str] = None) -> str:
        """Export instance data to a CSV file.

        Args:
            instances: List of instance data dictionaries
            output_file: Optional path to the output CSV file. If not provided,
                        a default name will be used.

        Returns:
            str: Path to the generated CSV file
        """
        if not output_file:
            output_file = "ec2_instances_export.csv"

        return cls.export_to_csv(instances, output_file)

    @classmethod
    def export_costs_to_csv(cls, costs: List[Dict[str, Any]],
                            output_file: Optional[str] = None) -> str:
        """Export cost data to a CSV file.

        Args:
            costs: List of cost data dictionaries
            output_file: Optional path to the output CSV file. If not provided,
                        a default name will be used.

        Returns:
            str: Path to the generated CSV file
        """
        if not output_file:
            output_file = "ec2_costs_export.csv"

        return cls.export_to_csv(costs, output_file)

    @classmethod
    def export_savings_to_csv(cls, savings: List[Dict[str, Any]],
                              output_file: Optional[str] = None) -> str:
        """Export savings analysis data to a CSV file.

        Args:
            savings: List of savings data dictionaries
            output_file: Optional path to the output CSV file. If not provided,
                        a default name will be used.

        Returns:
            str: Path to the generated CSV file
        """
        if not output_file:
            output_file = "ec2_savings_analysis_export.csv"

        return cls.export_to_csv(savings, output_file)
