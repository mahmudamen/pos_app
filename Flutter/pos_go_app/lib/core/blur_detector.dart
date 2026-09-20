import 'dart:typed_data';

import 'package:image/image.dart' as img;

// Blur detection for invoice photos before they are uploaded for OCR.
//
// Uses the variance of the Laplacian image-sharpness metric, computed on a
// downscaled grayscale copy so it stays fast on phones. A sharp photo of text
// has high edge energy (many crisp pixel transitions); a blurred one is
// mostly flat, so the variance is tiny. Pure Dart - no native vision SDKs.

/// The shutter-safety verdict for one captured image.
class BlurAssessment {
  const BlurAssessment({
    required this.score,
    required this.blurred,
  });

  /// Variance of the grayscale Laplacian over the downscaled image. Higher is
  /// sharper; typical sharp document photos land well above 100.
  final double score;

  /// True when [score] fell below the detector threshold.
  final bool blurred;
}

/// Variance-of-Laplacian blur detector with a sane default threshold.
///
/// The classifier is intentionally conservative: it warns (rather than only
/// fails) on weak text, and the UI offers a retake. [decode] is kept injectable
/// so tests can run the pure math on synthetic pixels without file I/O.
class BlurDetector {
  const BlurDetector({this.threshold = 40});

  /// Variance threshold below which an image counts as blurred.
  final double threshold;

  BlurAssessment assessBytes(Uint8List bytes) {
    final decoded = img.decodeImage(bytes);
    if (decoded == null) {
      throw const FormatException('decode failed (not a supported image)');
    }
    // Downscale the long edge to 256 px: plenty of edge information for the
    // Laplacian variance and ~100x faster than raw capture resolution. The
    // aspect ratio is kept so the blur metric is not distorted by stretching.
    final small = img.copyResize(decoded, width: 256, maintainAspect: true);
    final gray = img.grayscale(small);
    final score = varianceOfLaplacian(gray);
    return BlurAssessment(score: score, blurred: score < threshold);
  }

  /// Variance of the 3×3 Laplacian over [image]'s luminance channel.
  double varianceOfLaplacian(img.Image image) {
    final w = image.width;
    final h = image.height;
    if (w < 3 || h < 3) return 0;
    final lap = Float64List((w - 2) * (h - 2));
    var idx = 0;
    for (var y = 1; y < h - 1; y++) {
      for (var x = 1; x < w - 1; x++) {
        final center = _lum(image, x, y);
        final l = (4 * center -
                _lum(image, x - 1, y) -
                _lum(image, x + 1, y) -
                _lum(image, x, y - 1) -
                _lum(image, x, y + 1))
            .toDouble();
        lap[idx++] = l;
      }
    }
    if (idx == 0) return 0;
    var mean = 0.0;
    for (var i = 0; i < idx; i++) {
      mean += lap[i];
    }
    mean /= idx;
    var sumSq = 0.0;
    for (var i = 0; i < idx; i++) {
      final d = lap[i] - mean;
      sumSq += d * d;
    }
    return sumSq / idx;
  }

  num _lum(img.Image image, int x, int y) {
    final px = image.getPixel(x, y);
    return 0.299 * px.r + 0.587 * px.g + 0.114 * px.b;
  }
}