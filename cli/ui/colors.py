class Colors:
    """ANSI color codes for console output."""
    # Primary colors
    PRIMARY = '\033[38;5;39m'  # Soft blue
    SECONDARY = '\033[38;5;247m'  # Medium gray
    SUCCESS = '\033[38;5;34m'  # Soft green
    WARNING = '\033[38;5;208m'  # Orange
    DANGER = '\033[38;5;196m'  # Red
    CYAN = '\033[38;5;44m'  # Cyan

    # Text colors
    TEXT = '\033[38;5;250m'  # Light gray
    TEXT_BOLD = '\033[1;38;5;255m'  # White bold
    TEXT_MUTED = '\033[38;5;240m'  # Dark gray

    # Background colors
    BG_DARK = '\033[48;5;235m'  # Dark background

    # Utility
    RESET = '\033[0m'
    BOLD = '\033[1m'
    UNDERLINE = '\033[4m'

    # Aliases for compatibility
    HEADER = PRIMARY
    BLUE = PRIMARY
    GREEN = SUCCESS
    FAIL = DANGER
    ENDC = RESET

    @staticmethod
    def colorize(text: str, color: str, use_colors: bool = True) -> str:
        """Apply color to text if use_colors is True.

        Args:
            text: Text to colorize
            color: Color code to apply
            use_colors: Whether to apply colors

        Returns:
            Colored text if use_colors is True, else original text
        """
        if not use_colors or not color:
            return text
        return f"{color}{text}{Colors.RESET}"
