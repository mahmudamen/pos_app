import 'package:sales_pro/theme/app_theme.dart';
import 'package:flutter/material.dart';

class DataListItem extends StatelessWidget {
  final String title;
  final String? subtitle;
  final String? leading;
  final String? trailingTop;
  final String? trailingBottom;
  final VoidCallback? onTap;
  final Color? trailingColor;

  const DataListItem({
    super.key,
    required this.title,
    this.subtitle,
    this.leading,
    this.trailingTop,
    this.trailingBottom,
    this.onTap,
    this.trailingColor,
  });

  @override
  Widget build(BuildContext context) {
    return Card(
      margin: const EdgeInsets.symmetric(vertical: 4, horizontal: 2),
      elevation: 1,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
      child: InkWell(
        borderRadius: BorderRadius.circular(10),
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.all(12),
          child: Row(
            children: [
              // Leading (e.g. invoice number or ID)
              if (leading != null)
                Container(
                  width: 50,
                  alignment: Alignment.center,
                  child: Text(
                    leading!,
                    style: const TextStyle(fontWeight: FontWeight.bold),
                  ),
                ),

              if (leading != null) const SizedBox(width: 12),

              // Title + Subtitle
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      title,
                      style: const TextStyle(
                        fontWeight: FontWeight.w600,
                        fontSize: 15,
                      ),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      textAlign: TextAlign.right,
                    ),
                    if (subtitle != null) ...[
                      const SizedBox(height: 4),
                      Text(
                        subtitle!,
                        style: TextStyle(fontSize: 13, color: Colors.grey[600]),
                        textAlign: TextAlign.right,
                      ),
                    ],
                  ],
                ),
              ),

              // Trailing (price, balance, etc)
              if (trailingTop != null || trailingBottom != null)
                Container(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 12,
                    vertical: 6,
                  ),
                  decoration: BoxDecoration(
                    color: AppColors.tabActive.withOpacity(0.1),
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.end,
                    children: [
                      if (trailingTop != null)
                        Text(
                          trailingTop!,
                          style: TextStyle(
                            fontWeight: FontWeight.bold,
                            color: trailingColor ?? AppColors.tabActive,
                          ),
                        ),
                      if (trailingBottom != null) ...[
                        const SizedBox(height: 2),
                        Text(
                          trailingBottom!,
                          style: TextStyle(
                            fontSize: 12,
                            color: Colors.grey[600],
                          ),
                        ),
                      ],
                    ],
                  ),
                ),
            ],
          ),
        ),
      ),
    );
  }
}
