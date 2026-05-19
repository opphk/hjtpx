import UIKit

public class CaptchaImageView: UIView {
    private var imageView: UIImageView!
    private var sliderThumb: UIView!
    private var sliderTrack: UIView!

    private var initialTouchPoint: CGPoint = .zero
    private var sliderStartX: CGFloat = 0
    private var maxSliderX: CGFloat = 0

    public var onSliderMoved: ((Float) -> Void)?
    public var onSliderCompleted: ((Float) -> Void)?

    public override init(frame: CGRect) {
        super.init(frame: frame)
        setupViews()
    }

    required init?(coder: NSCoder) {
        super.init(coder: coder)
        setupViews()
    }

    private func setupViews() {
        backgroundColor = UIColor(white: 0.95, alpha: 1.0)
        layer.cornerRadius = 8
        clipsToBounds = true

        imageView = UIImageView()
        imageView.contentMode = .scaleAspectFill
        imageView.clipsToBounds = true
        addSubview(imageView)

        sliderTrack = UIView()
        sliderTrack.backgroundColor = UIColor(white: 0.8, alpha: 1.0)
        sliderTrack.layer.cornerRadius = 2
        addSubview(sliderTrack)

        sliderThumb = UIView()
        sliderThumb.backgroundColor = .white
        sliderThumb.layer.cornerRadius = 4
        sliderThumb.layer.shadowColor = UIColor.black.cgColor
        sliderThumb.layer.shadowOffset = CGSize(width: 0, height: 2)
        sliderThumb.layer.shadowOpacity = 0.3
        sliderThumb.layer.shadowRadius = 4

        let arrowImage = UIImageView()
        arrowImage.image = UIImage(systemName: "chevron.left.2")
        arrowImage.tintColor = UIColor.systemGray
        arrowImage.contentMode = .scaleAspectFit
        arrowImage.tag = 100
        sliderThumb.addSubview(arrowImage)

        addSubview(sliderThumb)

        let panGesture = UIPanGestureRecognizer(target: self, action: #selector(handlePan(_:)))
        sliderThumb.addGestureRecognizer(panGesture)
        sliderThumb.isUserInteractionEnabled = true
    }

    public override func layoutSubviews() {
        super.layoutSubviews()

        imageView.frame = CGRect(x: 0, y: 0, width: bounds.width, height: bounds.height - 50)
        sliderTrack.frame = CGRect(x: 10, y: bounds.height - 40, width: bounds.width - 20, height: 4)
        sliderThumb.frame = CGRect(x: 10, y: bounds.height - 50, width: 50, height: 30)

        maxSliderX = bounds.width - 60

        if let arrowImage = sliderThumb.viewWithTag(100) as? UIImageView {
            arrowImage.frame = CGRect(x: 15, y: 10, width: 20, height: 10)
        }
    }

    @objc private func handlePan(_ gesture: UIPanGestureRecognizer) {
        let translation = gesture.translation(in: self)

        switch gesture.state {
        case .began:
            initialTouchPoint = sliderThumb.center
            sliderStartX = sliderThumb.frame.origin.x
            generateHapticFeedback()

        case .changed:
            var newX = sliderStartX + translation.x
            newX = max(0, min(newX, maxSliderX))
            sliderThumb.frame.origin.x = newX

            let progress = Float(newX / maxSliderX)
            onSliderMoved?(progress)

        case .ended, .cancelled:
            let finalX = sliderThumb.frame.origin.x
            let progress = Float(finalX / maxSliderX)
            onSliderCompleted?(progress)
            generateHapticFeedback()

        default:
            break
        }
    }

    public func setBackgroundImage(_ image: UIImage?) {
        imageView.image = image
    }

    public func setSliderThumbImage(_ image: UIImage?) {
        if let image = image {
            if let arrowImage = sliderThumb.viewWithTag(100) as? UIImageView {
                arrowImage.image = image
            }
        }
    }

    public func resetSlider() {
        UIView.animate(withDuration: 0.3) {
            self.sliderThumb.frame.origin.x = 10
        }
    }

    private func generateHapticFeedback() {
        let generator = UIImpactFeedbackGenerator(style: .light)
        generator.impactOccurred()
    }

    public func showSuccess() {
        UIView.animate(withDuration: 0.3) {
            self.sliderThumb.backgroundColor = UIColor.systemGreen
        }
    }

    public func showFailure() {
        UIView.animate(withDuration: 0.3) {
            self.sliderThumb.backgroundColor = UIColor.systemRed
        }

        DispatchQueue.main.asyncAfter(deadline: .now() + 0.5) {
            self.resetSlider()
            UIView.animate(withDuration: 0.3) {
                self.sliderThumb.backgroundColor = .white
            }
        }
    }
}
